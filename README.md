# sendfile

Sendfile is useless utility to send files overinternet.

## Installing

```
~ $ go install github.com/vasyahuyasa/sendfile
```

## How to use

Reciver

```
~ $ sendfile -r 44667 media.iso
```

Sender

```
~ $ sendfile media.iso 173.194.73.94:44667
```
