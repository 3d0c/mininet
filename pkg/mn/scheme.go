package mn

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
)

// Scheme defenition
type Scheme struct {
	Switches []*Switch
	Hosts    []*Host
	pairs    map[string]bool
}

// Satisfies stringer interface
func (s Scheme) String() string {
	out, err := json.MarshalIndent(s, "", "      ")
	if err != nil {
		panic(err)
	}

	return string(out)
}

// NewScheme creates instance of the scheme
func NewScheme() *Scheme {
	return &Scheme{
		make([]*Switch, 0),
		make([]*Host, 0),
		make(map[string]bool),
	}
}

// NewSchemeFromJSON create scheme from json file
func NewSchemeFromJSON(fname string) (*Scheme, error) {
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	scheme := NewScheme()

	err = json.Unmarshal(data, scheme)
	if err != nil {
		return nil, err
	}

	return scheme, nil
}

// AddNode adds node into scheme
func (s *Scheme) AddNode(n interface{}) *Scheme {
	switch t := n.(type) {
	case *Switch:
		s.Switches = append(s.Switches, n.(*Switch))
	case *Host:
		s.Hosts = append(s.Hosts, n.(*Host))
	default:
		log.Printf("Wrong call, unknown type %s for %v\n", t, n)
	}

	return s
}

// GetNode returns Node depending on type
func (s *Scheme) GetNode(name string) (Node, bool) {
	if n, found := s.GetHost(name); found {
		return n, found
	}

	if n, found := s.GetSwitch(name); found {
		return n, found
	}

	return nil, false
}

// GetHost host getter
func (s *Scheme) GetHost(name string) (*Host, bool) {
	for _, host := range s.Hosts {
		if host.NodeName() == name {
			return host, true
		}
	}

	return nil, false
}

// GetSwitch switch getter
func (s *Scheme) GetSwitch(name string) (*Switch, bool) {
	for _, sw := range s.Switches {
		if sw.NodeName() == name {
			return sw, true
		}
	}

	return nil, false
}

// Nodes iterator
func (s *Scheme) Nodes() chan Node {
	yield := make(chan Node)

	go func() {
		for _, sw := range s.Switches {
			yield <- (Node)(sw)
		}

		for _, host := range s.Hosts {
			yield <- (Node)(host)
		}

		close(yield)
	}()

	return yield
}

// Export the scheme
func (s Scheme) Export() string {
	return s.String()
}

// Recover nodes scheme
func (s Scheme) Recover() error {
	for node := range s.Nodes() {
		switch t := node.(type) {
		case *Switch:
			s.recoverSwitchPorts(node.(*Switch))
		case *Host:
			s.recoverHostLinks(node.(*Host))
		default:
			log.Println("Unexpected type", t)
		}
	}

	for _, host := range s.Hosts {
		if err := host.recoverProcs(); err != nil {
			return err
		}
	}

	return nil
}

// Recover switch to host connectivity
func (s Scheme) recoverSwitchPorts(sw *Switch) error {
	for _, port := range sw.Ports {
		if port.Exists() {
			continue
		}

		peer, found := s.GetNode(port.Peer.NodeName)
		if !found {
			return fmt.Errorf("Can't find host %s", port.Peer.NodeName)
		}

		link := peer.GetLinks().LinkByPeer(port.Peer)
		pair := Pair{port, link}

		hash := port.Name + "-" + link.Name
		if s.pairs[hash] {
			log.Printf("Wrong scheme. Two identic pairs found: %v\n", pair)
			continue
		}

		// patch link
		if sw2, found := s.GetSwitch(peer.NodeName()); found {
			sw.AddPatchPort(pair.Left)
			sw2.AddPatchPort(pair.Right)
			continue
		}

		if err := pair.Create(); err != nil {
			return err
		}

		if err := sw.AddLink(pair.Left); err != nil {
			return err
		}

		_, err := pair.Up()
		if err != nil {
			return err
		}

		s.pairs[hash] = true
	}

	return nil
}

// recoverHostLinks host to host connectivity
func (s Scheme) recoverHostLinks(h *Host) error {
	for _, left := range h.Links {
		peer, found := s.GetHost(left.Peer.NodeName)
		if !found {
			continue
		}

		right := peer.Links.LinkByPeer(left.Peer)
		if right.NodeName == "" {
			// nothing found
			// @todo-maybe return (Link, bool) form LinkByPeer
			continue
		}

		pair := Pair{left, right}

		hash := left.NodeName + left.Name + right.NodeName + right.Name
		if s.pairs[hash] {
			continue
		}

		if err := pair.Create(); err != nil {
			return err
		}

		_, err := pair.Up()
		if err != nil {
			return err
		}

		h.AddLink(left)

		h2, found := s.GetHost(right.NodeName)
		if !found {
			return fmt.Errorf("Can't find host node %s", right.NodeName)
		}

		h2.AddLink(right)

		s.pairs[hash] = true
	}

	return nil
}

// Release nodes
func (s *Scheme) Release() {
	for node := range s.Nodes() {
		node.Release()
	}
}
