# mixed-socks
support socks4, socks4a, socks5, socks5h, http proxy all in one

## Help

```bash
# ./mixed-socks --help                                                                                                                                              ──(Sun,Sep04)─┘
Support socks4, socks4a, socks5, socks5h, http proxy all in one

Usage:
  ./mixed-socks [flags]

Flags:
  -c, --config string   config file path (default "config.yaml")
  -h, --help            help for ./mixed-socks
```

## Build

```bash
git clone https://github.com/xmapst/mixed-socks.git
cd mixed-socks
go mod tidy
go build -ldflags "-w -s" cmd/mixed-socks.go
```