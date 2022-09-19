package common

import "github.com/rs/xid"

func GUID() string {
	return xid.New().String()
}
