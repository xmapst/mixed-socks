# mixed-socks
support socks4, socks4a, socks5, socks5h, http proxy all in one

## Help

```bash
# ./mixed-socks -h
Usage of ./mixed-socks:
  -c string
        specify configuration file
  -v    show current version
```

## Build

```bash
git clone https://github.com/xmapst/mixed-socks.git
cd mixed-socks
make
```

## Config example
```yaml
# Support socks4, socks4a, socks5, socks5h, http proxy all in one
# HTTP(S) and SOCKS4(A)/SOCKS5 server on the same listening address
Inbound:
  Listen: 0.0.0.0
  Port: 8090

# Outbound settings
# This section is optional.
Outbound:
  # interface name
  Interface: eth0
  # fwmark on Linux only
  RoutingMark: 6666

# Controller settings
# This section is optional.
# RESTful web API listening address
Controller:
  Enable: true
  Listen: 0.0.0.0
  Port: 8080
  Secret: 123456

# default output to STDOUT when filename is empty
Log:
  # info / warning / error / debug / silent
  Level: info
  MaxBackups: 7
  MaxSize: 50
  MaxAge: 28
  Compress: true
  Filename: /var/log/mixed-socks/mixed-socks.log

# Auth settings
# This section is optional.
# authentication of local HTTP(S) and SOCKS4(A)/SOCKS5 server
Auth:
  "user1": pass1

# WhiteList settings
# This section is optional.
# whiteList of local HTTP(S) and SOCKS4(A)/SOCKS5 server
WhiteList:
  - 10.0.0.0/8
  - 172.16.0.0/16
  - 192.168.0.0/24

# Hosts settings
# This section is optional.
# Static hosts for DNS server and connection establishment (like /etc/hosts)
#
# Wildcard hostnames are supported (e.g. *.dev, *.foo.*.example.com)
# Non-wildcard domain names have a higher priority than wildcard domain names
# e.g. foo.example.com > *.example.com > .example.com
# P.S. +.foo.com equals to .foo.com and foo.com
Hosts:
  '*.dev': 127.0.0.1
  'alpha.dev': '::1'

# DNS server settings
# This section is optional. When not present, the DNS server will be disabled.
DNS:
  Enable: true
  Listen: 0.0.0.0
  Port: 53
  # Supports UDP, TCP, DoT, DoH. You can specify the port to connect to.
  # All DNS questions are sent directly to the nameserver, without proxies
  # involved. Answers the DNS question with the first result gathered.
  Nameservers:
    - 114.114.114.114 # default value
    - 8.8.8.8 # default value
    - tls://dns.rubyfish.cn:853 # DNS over TLS
    - https://1.1.1.1/dns-query # DNS over HTTPS
    - dhcp://en0 # dns from dhcp
```