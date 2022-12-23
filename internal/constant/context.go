package constant

import (
	"github.com/gofrs/uuid"
	"net"
)

type PlainContext interface {
	ID() uuid.UUID
}

type ConnContext interface {
	PlainContext
	Metadata() *Metadata
	Conn() net.Conn
}

type PacketConnContext interface {
	PlainContext
	Metadata() *Metadata
	PacketConn() net.PacketConn
}
