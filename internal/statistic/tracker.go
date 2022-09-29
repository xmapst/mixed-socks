package statistic

import (
	"go.uber.org/atomic"
	"net"
	"time"
)

type tracker interface {
	ID() string
	Close() error
}

type trackerInfo struct {
	UUID          string        `json:""`
	Metadata      *Metadata     `json:""`
	UploadTotal   *atomic.Int64 `json:""`
	DownloadTotal *atomic.Int64 `json:""`
	Start         time.Time     `json:""`
}

type TcpTracker struct {
	net.Conn `json:"-"`
	*trackerInfo
	manager *manager
}

func (tt *TcpTracker) ID() string {
	return tt.UUID
}

func (tt *TcpTracker) Read(b []byte) (int, error) {
	n, err := tt.Conn.Read(b)
	download := int64(n)
	tt.manager.PushDownloaded(download)
	tt.DownloadTotal.Add(download)
	return n, err
}

func (tt *TcpTracker) Write(b []byte) (int, error) {
	n, err := tt.Conn.Write(b)
	upload := int64(n)
	tt.manager.PushUploaded(upload)
	tt.UploadTotal.Add(upload)
	return n, err
}

func (tt *TcpTracker) Close() error {
	tt.manager.Leave(tt)
	return tt.Conn.Close()
}

func NewTCPTracker(id string, conn net.Conn, metadata *Metadata) *TcpTracker {
	t := &TcpTracker{
		Conn:    conn,
		manager: Manager,
		trackerInfo: &trackerInfo{
			UUID:          id,
			Start:         time.Now(),
			Metadata:      metadata,
			UploadTotal:   atomic.NewInt64(0),
			DownloadTotal: atomic.NewInt64(0),
		},
	}
	Manager.Join(t)
	return t
}
