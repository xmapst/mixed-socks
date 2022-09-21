package common

// SOCKS address types as defined in RFC 1928 section 5.
const (
	DefaultHost    = "0.0.0.0"
	DefaultPort    = 8090
	DefaultTimeout = "30s"

	AtypIPv4       = 1
	AtypDomainName = 3
	AtypIPv6       = 4
)
