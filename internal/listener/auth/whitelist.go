package auth

import "github.com/xmapst/mixed-socks/internal/component/auth"

var whitelist auth.Whitelist

func Whitelist() auth.Whitelist {
	return whitelist
}

func SetWhitelist(au auth.Whitelist) {
	whitelist = au
}
