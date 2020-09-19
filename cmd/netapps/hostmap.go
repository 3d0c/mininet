package netapps

import (
	"net"
	"sync"
)

// Host definition
type Host struct {
	mac  net.HardwareAddr
	port uint16
}

// Hostmap definition
type Hostmap struct {
	byMacIP map[string]map[string]Host
	byMac   map[string]Host
	sync.RWMutex
}

// NewHostMap creates new Hostmap instance
func NewHostMap() *Hostmap {
	return &Hostmap{
		byMacIP: make(map[string]map[string]Host),
		byMac:   make(map[string]Host),
	}
}

// Host return host from map
func (hm *Hostmap) Host(v ...interface{}) (h Host, ok bool) {
	hm.RLock()
	defer hm.RUnlock()

	switch len(v) {
	case 0:
		panic("Wrong call, expected at least one argument to HostMap.Host(...)")

	case 1:
		mac, ok := v[0].(net.HardwareAddr)
		if !ok {
			panic("Expected net.HardwareAddr")
		}

		h, ok := hm.byMac[mac.String()]
		return h, ok

	case 2:
		dpid, ok := v[0].(net.HardwareAddr)
		if !ok {
			panic("First argument expected to be net.HardwareAddr")
		}

		ip, ok := v[1].(net.IP)
		if !ok {
			panic("Second argument expected to be net.IP")
		}

		h, ok := hm.byMacIP[dpid.String()][ip.String()]
		return h, ok
	}

	return
}

// Add populates map
func (hm *Hostmap) Add(v ...interface{}) {
	hm.RLock()
	defer hm.RUnlock()

	switch len(v) {
	case 0:
		panic("Wrong call, expected at least one argument to HostMap.Host(...)")

	case 2:
		mac, ok := v[0].(net.HardwareAddr)
		if !ok {
			panic("Expected first argument to be a net.HardwareAddr")
		}

		port, ok := v[1].(uint16)
		if !ok {
			panic("Expected second argument to be an uint16")
		}

		hm.byMac[mac.String()] = Host{mac, port}
		return

	case 3:
		dpid, ok := v[0].(net.HardwareAddr)
		if !ok {
			panic("Expected first argument to be a net.HardwareAddr")
		}

		ip, ok := v[1].(net.IP)
		if !ok {
			panic("Expected first argument to be a net.HardwareAddr")
		}

		h, ok := v[2].(Host)
		if !ok {
			panic("Expected third argument to be a Host")
		}

		if _, found := hm.byMacIP[dpid.String()]; !found {
			hm.byMacIP[dpid.String()] = make(map[string]Host)
		}

		hm.byMacIP[dpid.String()][ip.String()] = h

		return

	default:
		panic("Wrong call, expected 2 or three arguments")
	}

}

// Dpid dpid to ip
func (hm *Hostmap) Dpid(dpid net.HardwareAddr) (map[string]Host, bool) {
	iptoHost, found := hm.byMacIP[dpid.String()]
	return iptoHost, found
}
