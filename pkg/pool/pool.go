package pool

import (
	"crypto/rand"
	"fmt"
	"net"
)

// Pool is structure which contains cached IPNet and pool of adresses
type Pool struct {
	cache   map[string]*net.IPNet
	pool    map[string]bool
	created bool
	preset  bool
}

var instance Pool

// ThePool is a singleton, wwhich creates a pool of IPs based on provided CIDR
func ThePool(args ...interface{}) Pool {
	if len(args) > 0 && !instance.created {
		ip, ipnet, err := net.ParseCIDR(args[0].(string))
		if err != nil {
			panic(err)
		}

		key := ip.Mask(ipnet.Mask)
		instance = newPool()
		instance.cache[key.String()] = ipnet
		instance.preset = true
	}

	if instance.created == false {
		instance = newPool()
	}

	return instance
}

func newPool() Pool {
	return Pool{
		cache:   make(map[string]*net.IPNet),
		pool:    make(map[string]bool),
		created: true,
	}
}

// NextCidr returns the next cidr
func (p Pool) NextCidr(args ...interface{}) string {
	var cidr string

	if len(args) == 1 {
		cidr = args[0].(string)
	} else {
		for _, k := range p.cache {
			ipnet := p.cache[k]
			cidr = ipnet.String()
			break
		}
	}

	ip, ipnet, key, err := p.get(cidr)
	if err != nil {
		panic(err)
	}

	inc := func(ip net.IP) {
		for j := len(ip) - 1; j >= 0; j-- {
			ip[j]++
			if ip[j] > 0 {
				break
			}
		}
	}

	inc(ip)
	if !ipnet.Contains(ip) {
		ip, ipnet, _ = net.ParseCIDR(ipnet.String())
	}

	ipnet.IP = ip

	if _, found := p.cache[key]; !found {
		p.cache[key] = &net.IPNet{}
	}

	p.cache[key] = ipnet

	return ipnet.String()
}

// NextAddr returns next address
func (p Pool) NextAddr(args ...interface{}) string {
	var addr, netmask string
	var ipnet *net.IPNet

	switch len(args) {
	case 2:
		addr = args[0].(string)
		netmask = args[1].(string)
		ipnet = &net.IPNet{
			IP:   net.ParseIP(addr),
			Mask: net.IPMask(net.ParseIP(netmask)),
		}

	case 0:
		for _, k := range p.cache {
			ipnet = p.cache[k]
			break
		}
	default:
		return ""
	}

	ip, _, _ := net.ParseCIDR(p.NextCidr(ipnet.String()))

	return ip.String()
}

// NextMac returns next MAC
func (p Pool) NextMac(mac string) string {
	hw, err := net.ParseMAC(mac)
	if err != nil {
		return ""
	}

	buf := make([]byte, 6)

	_, err = rand.Read(buf)
	if err != nil {
		return ""
	}

	buf[0] |= 2
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", hw[0], hw[1], hw[2], buf[3], buf[4], buf[5])
}

func (p Pool) get(cidr string) (net.IP, *net.IPNet, string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)

	// p is because inventory nic's contains cidr as a an ip address and
	// we can not use it as a keys in cache map. So we have to convert them
	// to the mask. E.g. cidr 10.200.22.31/16 becomes 10.200.0.0, and we use
	// it as a key.
	ip = ip.Mask(ipnet.Mask)

	if c, found := p.cache[ip.String()]; found {
		return c.IP, c, ip.String(), nil
	}

	return ip, ipnet, ip.String(), err
}

// Preset method is a wrapper for unexported field
func (p Pool) Preset() bool {
	return p.preset
}
