package listener

import (
	"github.com/sirupsen/logrus"
	"github.com/xmapst/mixed-socks/internal/adapter/inbound"
	"github.com/xmapst/mixed-socks/internal/constant"
	"github.com/xmapst/mixed-socks/internal/listener/mixed"
	"github.com/xmapst/mixed-socks/internal/listener/socks"
	"net"
	"sync"
)

var (
	mixedListener  *mixed.Listener
	mixedUDPLister *socks.UDPListener

	// lock for recreate function
	mixedMux sync.Mutex
)

type Ports struct {
	Port int `json:"port"`
}

func ReCreateMixed(addr string, tcpIn chan<- constant.ConnContext, udpIn chan<- *inbound.PacketAdapter) {
	mixedMux.Lock()
	defer mixedMux.Unlock()

	var err error
	defer func() {
		if err != nil {
			logrus.Errorln("Start Mixed(http+socks) server error: ", err.Error())
		}
	}()

	shouldTCPIgnore := false
	shouldUDPIgnore := false

	if mixedListener != nil {
		if mixedListener.RawAddress() != addr {
			_ = mixedListener.Close()
			mixedListener = nil
		} else {
			shouldTCPIgnore = true
		}
	}
	if mixedUDPLister != nil {
		if mixedUDPLister.RawAddress() != addr {
			_ = mixedUDPLister.Close()
			mixedUDPLister = nil
		} else {
			shouldUDPIgnore = true
		}
	}

	if shouldTCPIgnore && shouldUDPIgnore {
		return
	}

	if portIsZero(addr) {
		return
	}

	mixedListener, err = mixed.New(addr, tcpIn)
	if err != nil {
		return
	}

	mixedUDPLister, err = socks.NewUDP(addr, udpIn)
	if err != nil {
		_ = mixedListener.Close()
		return
	}

	logrus.Infof("Mixed(http+socks) proxy listening at: %s", mixedListener.Address())
}

func portIsZero(addr string) bool {
	_, port, err := net.SplitHostPort(addr)
	if port == "0" || port == "" || err != nil {
		return true
	}
	return false
}
