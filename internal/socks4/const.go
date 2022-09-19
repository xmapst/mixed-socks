package socks4

import "errors"

const (
	Version = 4
)

type Command = uint8

const (
	CmdConnect Command = 1
	CmdBind    Command = 2
)

var cmdMap = map[Command]string{
	CmdConnect: "CONNECT",
	CmdBind:    "BIND",
}

type Code = uint8

const (
	RequestGranted          Code = 90
	RequestRejected         Code = 91
	RequestIdentdFailed     Code = 92
	RequestIdentdMismatched Code = 93
)

var (
	errVersionMismatched   = errors.New("version code mismatched")
	errCommandNotSupported = errors.New("command not supported")
	errIPv6NotSupported    = errors.New("IPv6 not supported")

	ErrRequestRejected         = errors.New("request rejected or failed")
	ErrRequestIdentdFailed     = errors.New("request rejected because SOCKS server cannot connect to identd on the client")
	ErrRequestIdentdMismatched = errors.New("request rejected because the client program and identd report different user-ids")
	ErrRequestUnknownCode      = errors.New("request failed with unknown code")
)
