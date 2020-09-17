package mn

import (
	"errors"
	"fmt"
	"net"
	"reflect"

	"github.com/3d0c/mininet/pkg/pool"
)

// Pair defenition
type Pair struct {
	Left  Link
	Right Link
}

// Route definition
type Route struct {
	Dst string
	Gw  string
}

// Peer definition
type Peer struct {
	Name     string
	IfName   string
	NodeName string
}

// Link definition
type Link struct {
	Cidr      string
	HwAddr    string
	Name      string
	NodeName  string
	NetNs     string
	State     string
	Routes    []Route
	PeerName  string
	Peer      Peer
	patch     bool
	ForceRoot bool `json:"-"`
}

const noip = "noip"

// NewLink (n1, n2, [link1 properties, link2 properties])
// property which isn't specified will be generated
func NewLink(left, right Node, refs ...Link) Pair {
	result := Pair{}

	switch len(refs) {
	case 1:
		result = Pair{Left: refs[0], Right: Link{}}
	case 2:
		result = Pair{Left: refs[0], Right: refs[1]}
	default:
		break
	}

	if reflect.TypeOf(left).Elem() == reflect.TypeOf(right).Elem() && reflect.TypeOf(left).Elem() == reflect.TypeOf(Switch{}) {
		result.Left = result.Left.SetNodeName(left).SetName(left, "pp").SetPatch()
		result.Right = result.Right.SetNodeName(right).SetName(right, "pp").SetPatch()
	} else {
		result.Left = result.Left.SetCidr().SetHwAddr().
			SetNetNs(left).SetName(right, "eth").SetNodeName(left).SetState("DOWN").SetRoute()

		result.Right = result.Right.SetCidr().SetHwAddr().
			SetNetNs(right).SetName(right, "eth").SetNodeName(right).SetState("DOWN").SetRoute()
	}

	result.Left = result.Left.SetPeer(result.Right)
	result.Right = result.Right.SetPeer(result.Left)

	return result
}

// type link Link
//
// func (pr *Link) UnmarshalJSON(b []byte) error {
// 	l := &link{}

// 	if err := json.Unmarshal(b, l); err != nil {
// 		return err
// 	}

// 	*pr = *(*Link)(l)
// 	return nil
// }

// ByNodeName gets Link by nodename
func (pr Pair) ByNodeName(n Node) Link {
	if pr.Left.NodeName == n.NodeName() {
		return pr.Left
	}

	return pr.Right
}

// Create creates link between veth pair
func (pr Pair) Create() error {
	command := []string{"link", "add", "name", pr.Left.Name, "type", "veth", "peer", "name", pr.Right.Name}

	if pr.Right.NetNs != "" {
		command = append(command, "netns", pr.Right.NetNs)
	}

	if out, err := RunCommand("ip", command...); err != nil {
		return fmt.Errorf("Error: %v, output: %s", err, out)
	}

	if pr.Left.NetNs != "" {
		if err := pr.Left.MoveToNs(pr.Left.NetNs); err != nil {
			return err
		}
	}

	return nil
}

// Up sets pair on
func (pr Pair) Up() (Pair, error) {
	if pr.Left.patch {
		return pr, nil
	}

	if err := pr.Left.ApplyCidr(); err != nil {
		return pr, errors.New(fmt.Sprint("Unable to Left.ApplyCidr, error:", err))
	}

	if err := pr.Right.ApplyCidr(); err != nil {
		return pr, errors.New(fmt.Sprint("Unable to Right.ApplyCidr, error:", err))
	}

	if err := pr.Left.Up(); err != nil {
		return pr, errors.New(fmt.Sprint("Unable to Left.Up(), error:", err))
	}

	if err := pr.Right.Up(); err != nil {
		return pr, errors.New(fmt.Sprint("Unable to Right.Up(), error:", err))
	}

	if err := pr.Right.ApplyRoutes(); err != nil {
		return pr, errors.New(fmt.Sprint("Unable to ApplyRoutes(), error:", err))
	}

	pr.Left = pr.Left.SetState("UP")
	pr.Right = pr.Right.SetState("UP")

	fmt.Println("[Link]", pr.Left.NodeName, pr.Left.Name, pr.Left.Cidr, "<--->", pr.Right.NodeName, pr.Right.Name, pr.Right.Cidr)

	return pr, nil
}

// Release pair
func (pr Pair) Release() {
	pr.Left.Release()
	pr.Right.Release()
}

// IsPatch check wheter it's patch or not
func (pr Pair) IsPatch() bool {
	return pr.Left.patch
}

// Release the link
func (l Link) Release() {
	command := []string{"ip", "link", "delete", pr.Name}

	if l.NetNs != "" {
		command = append([]string{"ip", "netns", "exec", l.NetNs}, command...)
	}

	RunCommand(command[0], command[1:]...)
}

// ApplyMac applies MAC address
func (l Link) ApplyMac() error {
	if out, err := RunCommand("ip", "link", "set", "dev", pr.Name, "address", l.HwAddr); err != nil {
		return fmt.Errorf("Error: %v, output: %s", err, out)
	}

	return nil
}

// Up sets link to on
func (l Link) Up() error {
	command := []string{"ip", "link", "set", l.Name, "up"}

	if l.NetNs != "" {
		command = append([]string{"ip", "netns", "exec", l.NetNs}, command...)
	}

	if out, err := RunCommand(command[0], command[1:]...); err != nil {
		return fmt.Errorf("Error: %v, output: %s", err, out)
	}

	l.State = "UP"

	return nil
}

// ApplyCidr applies CIDR to the link
func (l Link) ApplyCidr() error {
	if _, _, err := net.ParseCIDR(l.Cidr); err != nil {
		// omit setting ip, by passing some garbage to input
		return nil
	}

	command := []string{"ip", "addr", "add", l.Cidr, "dev", l.Name}

	if l.NetNs != "" {
		command = append([]string{"ip", "netns", "exec", l.NetNs}, command...)
	}

	if out, err := RunCommand(command[0], command[1:]...); err != nil {
		return fmt.Errorf("Error: %v, output: %s", err, out)
	}

	return nil
}

// ApplyRoutes adds routing rule to the link
func (l Link) ApplyRoutes() error {
	for _, route := range l.Routes {
		commands := []string{"route", "add", "-net", route.Dst, "gw", route.Gw}
		if l.NetNs != "" {
			commands = append([]string{"ip", "netns", "exec", l.NetNs}, commands...)
		}

		out, err := RunCommand(commands[0], commands[1:]...)
		if err != nil {
			return fmt.Errorf("Error: %v, output: %s", err, out)
		}
	}

	return nil
}

// Exists checks wheter link exist or not
func (l Link) Exists() bool {
	_, err := RunCommand("ip", "link", "show", l.Name)
	return err == nil
}

// MoveToNs moves link to another network namespace
func (l Link) MoveToNs(netns string) error {
	if out, err := RunCommand("ip", "link", "set", l.Name, "netns", netns); err != nil {
		return fmt.Errorf("Error: %v, output: %s", err, out)
	}

	return nil
}

// SetCidr sets next CIDR from the pool to the link
func (l Link) SetCidr() Link {
	if l.Cidr == "" {
		l.Cidr = pool.ThePool().NextCidr()
	}

	return l
}

// SetHwAddr sets MAC address to the link
func (l Link) SetHwAddr() Link {
	if l.HwAddr == "" {
		l.HwAddr = pool.ThePool().NextMac(firstrealhw().HardwareAddr.String())
	}

	return l
}

// SetName sets the name,
// interface pairs naming rules:
//   left (host node)          {peer_host}-{prefix}X
//   right (namespaced node)   ethX
func (l Link) SetName(n Node, prefix string) Link {
	if l.Name != "" {
		return l
	}

	if l.NetNs != "" {
		l.Name = fmt.Sprintf("veth%d", n.LinksCount())
	} else {
		l.Name = fmt.Sprintf("%s-%s%d", n.NodeName(), prefix, n.LinksCount())
	}

	return l
}

// SetNodeName for link
func (l Link) SetNodeName(n Node) Link {
	if l.NodeName == "" {
		l.NodeName = n.NodeName()
	}

	return s
}

// SetNetNs sets network name space
func (l Link) SetNetNs(n Node) Link {
	if l.NetNs == "root" {
		l.NetNs = ""
		return l
	}

	if l.NetNs == "" && n.NetNs() != nil {
		l.NetNs = n.NetNs().Name()
	}

	return l
}

// SetState sets link state
func (l Link) SetState(s string) Link {
	l.State = s
	return l
}

// SetRoute tmp
func (l Link) SetRoute() Link {
	return l
}

// SetPeer sets peer to the link
func (l Link) SetPeer(in Link) Link {
	l.Peer.IfName = in.Name
	l.Peer.NodeName = in.NodeName
	l.Peer.Name = in.Name

	return l
}

// SetPatch sets patch to the link
func (l Link) SetPatch() Link {
	if !l.ForceRoot {
		l.patch = true
	}

	return l
}

// IP returns IP from link's CIDR
func (l Link) IP() string {
	ip, _, _ := net.ParseCIDR(l.Cidr)
	return ip.String()
}

// Links is a set of Links
type Links []Link

// LinkByPeer search link by peer
func (ls Links) LinkByPeer(peer Peer) Link {
	for _, link := range ls {
		if link.NodeName == peer.NodeName && link.Name == peer.IfName {
			return link
		}
	}

	return Link{}
}
