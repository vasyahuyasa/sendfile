package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

const sendBufSize = 1024 * 1024

type requestHeader struct {
	filename string
	from     int64
}

type responseHeader struct {
	filename string
	from     int64
	size     int64
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		usage()
		return
	}

	if os.Args[1] == "-r" {
		strPort := os.Args[2]
		port, err := strconv.ParseUint(strPort, 10, 32)
		if err != nil {
			usage()
			return
		}
		addr := fmt.Sprintf(":%d", port)

		var f *os.File
		if len(os.Args) >= 4 {
			f, err = os.OpenFile(os.Args[3], os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
			if err != nil {
				log.Printf("can not open %q: %v", os.Args[3], err)
				return
			}
		} else {
			f = os.Stdout
		}
		defer f.Close()

		l, err := net.Listen("tcp", addr)
		if err != nil {
			log.Printf("can not listen for incoming connections on %q: %v", addr, err)
			return
		}

		conn, err := l.Accept()
		if err != nil {
			log.Printf("can not accept connection: %v", err)
			return
		}

		tcpConn, ok := conn.(*net.TCPConn)
		if !ok {
			if !ok {
				log.Printf("connection is not TCP but must be TCP")
				return
			}
		}
		defer tcpConn.Close()

		err = recive(f, tcpConn)
		if err != nil {
			log.Printf("can not recive remote file")
			return
		}
	} else {
		path, addr := os.Args[1], os.Args[2]
		f, err := os.Open(path)
		if err != nil {
			log.Printf("can not open %q: %v", path, err)
			return
		}
		defer f.Close()

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Printf("can not connect to %q: %v", addr, err)
			return
		}
		defer conn.Close()

		tcpConn, ok := conn.(*net.TCPConn)
		if !ok {
			log.Printf("connection is not TCP but must be TCP")
			return
		}

		h, err := readRequestHeader(tcpConn)
		if err != nil {
			log.Printf("can not read request header: %v", err)
			return
		}

		_, err = f.Seek(h.from, io.SeekStart)
		if err != nil {
			log.Printf("can not seek to position %d: %v", h.from, err)
			return
		}

		err = send(f, h.from, tcpConn)
		if err != nil {
			log.Printf("can not send file %q: %v", path, err)
			return
		}
	}
}

func (h responseHeader) asBytes() []byte {
	return []byte(fmt.Sprintf("FILE %s\nFROM %d\nSIZE %d\n", h.filename, h.from, h.size))
}

func (h requestHeader) asBytes() []byte {
	return []byte(fmt.Sprintf("FILE %s\nFROM %d\n", h.filename, h.from))
}

func readRequestHeader(r io.Reader) (requestHeader, error) {
	var h requestHeader
	_, err := fmt.Fscanf(r, "FILE %s\nFROM %d\n", &h.filename, &h.from)
	if err != nil {
		return requestHeader{}, err
	}

	return h, nil
}

func readResponseHeader(r io.Reader) (responseHeader, error) {
	var h responseHeader
	_, err := fmt.Fscanf(r, "FILE %s\nFROM %d\nSIZE %d\n", &h.filename, &h.from, &h.size)
	if err != nil {
		return responseHeader{}, err
	}

	return h, nil
}

func send(f *os.File, from int64, conn *net.TCPConn) error {
	_, filename := filepath.Split(f.Name())
	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("can not read file stat: %w", err)
	}

	remain := stat.Size() - from

	h := responseHeader{
		filename: filename,
		from:     from,
		size:     remain,
	}
	_, err = conn.Write(h.asBytes())
	if err != nil {
		return fmt.Errorf("can not send file name: %w", err)
	}

	for remain > 0 {
		n, err := conn.ReadFrom(f)
		if err != nil {
			return fmt.Errorf("can not do sendfile: %w", err)
		}
		remain -= n
	}

	return nil
}

func recive(f *os.File, conn *net.TCPConn) error {
	_, filename := filepath.Split(f.Name())
	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("can not read file stat: %w", err)
	}

	from := stat.Size()
	h := requestHeader{
		filename: filename,
		from:     from,
	}

	_, err = conn.Write(h.asBytes())
	if err != nil {
		return fmt.Errorf("can not send request header: %w", err)
	}

	resp, err := readResponseHeader(conn)
	if err != nil {
		return fmt.Errorf("can not read response header: %w", err)
	}

	if resp.filename != filename && from != 0 {
		return fmt.Errorf("request file name mistmatch want %q but get %q", filename, resp.filename)
	}

	_, err = io.Copy(f, conn)
	if err != nil {
		return fmt.Errorf("can not recive remote file: %w", err)
	}

	return nil
}

func usage() {
	fmt.Println("sendfile <file> <address> - send file to specified address\nsendfile -r <port> [file]- recive file\nsendfile -h, --help - this help")
}
