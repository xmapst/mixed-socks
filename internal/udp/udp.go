package udp

import (
	"encoding/binary"
	"github.com/sirupsen/logrus"
	"github/xmapst/mixed-socks/internal/common"
	"net"
	"strconv"
	"strings"
	"time"
)

type udp struct {
	conn      *net.UDPConn
	srcUdpMap SrcUdpMap
}

func Listen(conn *net.UDPConn) {
	u := &udp{
		conn: conn,
	}
	go u.timeout()
	for {
		var data = make([]byte, 8192)
		n, srcAddr, err := conn.ReadFromUDP(data)
		if err != nil {
			if strings.Contains(err.Error(), net.ErrClosed.Error()) {
				break
			}
			logrus.Errorln("READ error", err)
			continue
		}
		if n <= 0 {
			continue
		}
		logrus.Infof("[%v]:", srcAddr)
		go u.handleUdpPacket(srcAddr, data[:n])
	}
}

func (u *udp) timeout() {
	tick := time.Tick(time.Second * 100)
	for {
		select {
		case <-tick:
			u.srcUdpMap.timeout()
		}
	}

}

/**
  +----+------+------+----------+----------+----------+
   |RSV | FRAG | ATYP | DST.ADDR | DST.PORT |   DATA   |
   +----+------+------+----------+----------+----------+
   | 2  |  1   |  1   | Variable |    2     | Variable |
   +----+------+------+----------+----------+----------+

  The fields in the UDP request header are:

       o  RSV  Reserved X'0000'
       o  FRAG    Current fragment number
       o  ATYP    address type of following addresses:
          o  IP V4 address: X'01'
          o  DOMAINNAME: X'03'
          o  IP V6 address: X'04'
       o  DST.ADDR       desired destination address
       o  DST.PORT       desired destination port
       o  DATA     user data
*/

func (u *udp) handleUdpPacket(srcAddr *net.UDPAddr, message []byte) {
	logrus.Infoln(srcAddr.String() + " send udp package!")
	length := len(message)
	index := 3
	if length < index {
		logrus.Errorln("error package")
		return
	}
	if message[0] != 0x00 && message[1] != 0x00 {
		logrus.Errorln("rev failed not 0x0000")
		return
	}
	if message[2] != 0x00 {
		logrus.Errorln("FRAG  not support")
		return
	}
	atype := message[3]
	index = 4
	var addr = ""
	if atype == common.AtypIPv4 {
		index += 4
		addr = net.IP(message[index-4 : index]).String()
	} else if atype == common.AtypDomainName {
		addrLen := int(message[index])
		index += 1
		addr = string(message[index : index+addrLen])
		index += addrLen
	} else if atype == common.AtypIPv6 {
		addr = net.IP(message[index : index+16]).String()
		index += 16
	}
	port := binary.BigEndian.Uint16(message[index : index+2])
	index += 2
	data := message[index:]
	originHeader := message[0:index]
	u.handleUdpPacket2(srcAddr, addr, port, data, originHeader)

}

func (u *udp) handleUdpPacket2(srcAddr *net.UDPAddr, dstAddr string, port uint16, message []byte, originHeader []byte) {
	srcUdpInfo := u.srcUdpMap.get(srcAddr)
	laddr := srcUdpInfo.localAddr
	var destAddr *net.UDPAddr
	ua := dstAddr + ":" + strconv.Itoa(int(port))
	remoteConn := srcUdpInfo.getRemoteConn(ua)
	if remoteConn == nil {
		destAddr, _ = net.ResolveUDPAddr("udp", ua)
		udpCon, err := net.DialUDP("udp", laddr, destAddr)
		if err != nil {
			logrus.Warningln("error connect " + dstAddr)
			return
		}
		remoteConn = udpCon
		srcUdpInfo.addRemoteConn(ua, remoteConn)
		if laddr == nil {
			srcUdpInfo.setLocalAddr(udpCon.LocalAddr().(*net.UDPAddr))
		}
		go u.handleRemoteRead(srcAddr, udpCon, originHeader, ua, srcUdpInfo)
	}
	_, err := remoteConn.Write(message)
	if err != nil {
		srcUdpInfo.deleteRemoteConn(ua)
		return
	}
	srcUdpInfo.active()
}

func (u *udp) handleRemoteRead(srcAddr *net.UDPAddr, udpCon *net.UDPConn,
	originHeader []byte, key string, info *SrcUdpInfo) {
	var b [65507]byte
	for {
		err := udpCon.SetReadDeadline(time.Now().Add(time.Second * 100))
		if err != nil {
			logrus.Warningln(err)
		}
		n, err := udpCon.Read(b[:])
		if err != nil {
			logrus.Warningln("udp read error==========", err)
			break
		}
		info.active()
		buf := append(originHeader, b[:n]...)
		_, err = u.conn.WriteToUDP(buf, srcAddr)
		if err != nil {
			logrus.Warningln(err)
		}
	}

	info.deleteRemoteConn(key)

}

type SrcUdpMap struct {
	associated map[string]*SrcUdpInfo // src string->src addr
}

func (u *SrcUdpMap) get(srcAddr *net.UDPAddr) *SrcUdpInfo {
	src := srcAddr.String()
	if u.associated[src] != nil {
		return u.associated[src]
	}
	r := &SrcUdpInfo{
		srcAddr:        srcAddr,
		lastActiveTime: time.Now(),
		localDestCon:   make(map[string]*net.UDPConn),
	}
	return r
}

func (u *SrcUdpMap) delete(srcAddr *net.UDPAddr) {
	src := srcAddr.String()
	delete(u.associated, src)
}

func (u *SrcUdpMap) timeout() {
	for k, v := range u.associated {
		if v.lastActiveTime.Add(time.Second * 100).Before(time.Now()) {
			delete(u.associated, k)
			v.Destroy()
			logrus.Warningln("delete" + k)
		}
	}
}

type SrcUdpInfo struct {
	srcAddr        *net.UDPAddr
	localAddr      *net.UDPAddr
	lastActiveTime time.Time
	localDestCon   map[string]*net.UDPConn //  dst -> conn
}

func (u *SrcUdpInfo) setLocalAddr(localAddr *net.UDPAddr) {
	u.localAddr = localAddr
	u.lastActiveTime = time.Now()

}

func (u *SrcUdpInfo) active() {
	u.lastActiveTime = time.Now()
}

func (u *SrcUdpInfo) deleteRemoteConn(remoteAddr string) {
	if c, ok := u.localDestCon[remoteAddr]; ok {
		err := c.Close()
		if err != nil {
			logrus.Warningln(err)
		}
		delete(u.localDestCon, remoteAddr)
	}
}

func (u *SrcUdpInfo) addRemoteConn(remoteAddr string, con *net.UDPConn) {
	u.localDestCon[remoteAddr] = con
}

func (u *SrcUdpInfo) getRemoteConn(remoteAddr string) *net.UDPConn {
	if c, ok := u.localDestCon[remoteAddr]; ok {
		return c
	}
	return nil
}

func (u *SrcUdpInfo) Destroy() {
	for _, v := range u.localDestCon {
		err := v.Close()
		if err != nil {
			logrus.Warningln(err)
		}
	}
}
