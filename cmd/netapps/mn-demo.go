package netapps

import (
	"log"
	"net"

	"github.com/3d0c/ogo/protocol/ofp10"
)

// NewDemoInstance ceates demo instance
func NewDemoInstance() interface{} {
	return new(DemoInstance)
}

// DemoInstance definition
type DemoInstance struct{}

// ConnectionUp logger
func (b *DemoInstance) ConnectionUp(dpid net.HardwareAddr) {
	log.Println("Switch connected:", dpid)
}

// ConnectionDown logger
func (b *DemoInstance) ConnectionDown(dpid net.HardwareAddr) {
	log.Println("Switch disconnected:", dpid)
}

// PacketIn processes input packet
func (b *DemoInstance) PacketIn(dpid net.HardwareAddr, pkt *ofp10.PacketIn) {
	log.Println("PacketIn message received from:", dpid, "len:", pkt.Len(), "datalen:", pkt.Data.Len(), "hwsrc:", pkt.Data.HWSrc, "hwdst:", pkt.Data.HWDst, pkt.Data.Ethertype)
}
