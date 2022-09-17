package socks5

const Version = 5

type Command = uint8

const (
	CmdConnect Command = 1
	CmdBind    Command = 2
	CmdUdp     Command = 3
)
