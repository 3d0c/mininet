package mn

import (
	"encoding/json"
	"fmt"
	"log"
)

// Switch model
type Switch struct {
	Name       string
	Ports      Links
	Controller string
}

// String implements Stringer interface
func (s Switch) String() string {
	out, err := json.MarshalIndent(s, "", "      ")
	if err != nil {
		panic(err)
	}

	return string(out)
}

// NewSwitch is a constructor for Switch model
func NewSwitch(name ...string) (*Switch, error) {
	s := &Switch{
		Name:  "",
		Ports: make(Links, 0),
	}

	if len(name) == 0 || name[0] == "" {
		s.Name = switchname()
	} else {
		s.Name = name[0]
	}

	if s.Exists() {
		return s, nil
	}

	if err := s.Create(); err != nil {
		return s, err
	}

	return s, nil
}

// UnmarshalJSON implements unmarshaller
func (s *Switch) UnmarshalJSON(b []byte) error {
	type tmp Switch
	t := tmp{}

	if err := json.Unmarshal(b, &t); err != nil {
		return err
	}

	s.Name = t.Name
	s.Ports = t.Ports
	if !s.Exists() {
		if err := s.Create(); err != nil {
			return err
		}
	}

	if s.Controller != "" {
		if err := s.SetController(s.Controller); err != nil {
			return err
		}
	}

	return nil
}

// Create creates switch
func (s *Switch) Create() error {
	out, err := RunCommand("ovs-vsctl", "add-br", s.Name)
	if err != nil {
		return fmt.Errorf("Error: %v, output: %s", err, out)
	}

	return nil
}

// Exists methods check whether Switch exists or not
func (s *Switch) Exists() bool {
	_, err := RunCommand("ovs-vsctl", "br-exists", s.Name)
	return err == nil
}

// AddLink adds link
func (s *Switch) AddLink(l Link) error {
	if l.patch {
		return s.AddPatchPort(l)
	}

	return s.AddPort(l)
}

// AddPort adds port to the link
func (s *Switch) AddPort(l Link) error {
	out, err := RunCommand("ovs-vsctl", "add-port", s.Name, l.Name)
	if err != nil {
		return fmt.Errorf("Error: %v, output: %s", err, out)
	}

	s.Ports = append(s.Ports, l)

	return nil
}

// AddPatchPort adds type to path
func (s *Switch) AddPatchPort(l Link) error {
	if out, err := RunCommand("ovs-vsctl", "add-port", s.NodeName(), l.Name); err != nil {
		return fmt.Errorf("Error: %v, output: %s", err, out)
	}

	if out, err := RunCommand("ovs-vsctl", "set", "interface", l.Name, "type=patch"); err != nil {
		return fmt.Errorf("Error: %v, output: %s", err, out)
	}

	if out, err := RunCommand("ovs-vsctl", "set", "interface", l.Name, "options:peer="+l.Peer.Name); err != nil {
		return fmt.Errorf("Error: %v, output: %s", err, out)
	}

	l = l.SetState("UP")

	s.Ports = append(s.Ports, l)

	return nil
}

// SetController sets Controller name and address
func (s *Switch) SetController(addr string) error {
	if out, err := RunCommand("ovs-vsctl", "set-controller", s.NodeName(), addr); err != nil {
		return fmt.Errorf("Error: %v, output: %s", err, out)
	}

	// if out, err := RunCommand("ovs-vsctl", "set", "bridge", s.NodeName(), "protocols=OpenFlow13"); err != nil {
	// 	return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
	// }

	// if out, err := RunCommand("ovs-vsctl", "set", "bridge", s.NodeName()); err != nil {
	// 	return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
	// }

	return nil
}

// Release removes bridge
func (s Switch) Release() error {
	out, err := RunCommand("ovs-vsctl", "del-br", s.Name)
	if err != nil {
		log.Println("Unable to delete bridge", s.Name, err, out)
	}

	return nil
}

// NodeName getter
func (s Switch) NodeName() string {
	return s.Name
}

// NetNs is a Network namespace getter
func (s Switch) NetNs() *NetNs {
	return nil
}

// LinksCount returns count of available links
func (s Switch) LinksCount() int {
	return len(s.Ports)
}

// GetCidr Link CIDR by peer getter
func (s Switch) GetCidr(peer Peer) string {
	return s.Ports.LinkByPeer(peer).Cidr
}

// GetHwAddr MAC address by peer getter
func (s Switch) GetHwAddr(peer Peer) string {
	return s.Ports.LinkByPeer(peer).HwAddr
}

// GetState state getter
func (s Switch) GetState(peer Peer) string {
	return s.Ports.LinkByPeer(peer).State
}

// GetLinks ports(links) getter
func (s Switch) GetLinks() Links {
	return s.Ports
}
