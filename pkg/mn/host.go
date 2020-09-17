package mn

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

// Host structure
type Host struct {
	Cgroup *Cgroup
	Name   string
	netns  *NetNs
	Links  Links
	Procs  Procs
}

// NewRouter creates a host instance with forwarding enabled
func NewRouter(name ...string) (*Host, error) {
	host, err := NewHost(name...)
	if err != nil {
		return nil, err
	}

	if err = host.enableForwarding(); err != nil {
		return nil, err
	}

	return host, nil
}

// NewHost create host instance
func NewHost(name ...string) (*Host, error) {
	host := &Host{
		Name:  "",
		Links: make(Links, 0),
	}

	if len(name) == 0 || name[0] == "" {
		host.Name = hostname(1024)
	} else {
		host.Name = name[0]
	}

	var err error

	if host.netns, err = NewNetNs(host.Name); err != nil {
		return nil, err
	}

	return host, nil
}

// String satisfies stringer interface
func (h Host) String() string {
	out, err := json.MarshalIndent(h, "", "      ")
	if err != nil {
		panic(err)
	}

	return string(out)
}

// UnmarshalJSON satisfies Mashaller
func (h *Host) UnmarshalJSON(b []byte) error {
	type tmp Host
	host := tmp{}

	if err := json.Unmarshal(b, &host); err != nil {
		return err
	}

	h.Name = host.Name
	h.Links = host.Links
	h.netns = &NetNs{name: host.Name}
	h.Procs = host.Procs
	h.Cgroup = host.Cgroup

	if !h.netns.Exists() {
		if err := h.NetNs().Create(); err != nil {
			return err
		}
	}

	if len(h.Links) > 1 {
		h.enableForwarding()
	}

	return nil
}

// RunProcess Exported wrapper for run process
func (h *Host) RunProcess(args ...string) (*Process, error) {
	p, err := h.runProcess(args...)
	if err != nil {
		return nil, err
	}

	h.Procs = append(h.Procs, p)
	return p, nil
}

func (h *Host) runProcess(args ...string) (*Process, error) {
	var command []string

	if h.Cgroup != nil {
		command = h.Cgroup.CgExecCommand()
	}

	ipCmd := FullPathFor("ip")
	if ipCmd == "" {
		return nil, errors.New("ip command not found the PATH")
	}

	if h.NetNs() != nil {
		command = append(command, []string{ipCmd, "netns", "exec", h.NetNs().Name()}...)
	}

	command = append(command, args...)

	p := &Process{Command: args[0], Args: args[1:]}

	fname := fmt.Sprintf("/tmp/output.%d", time.Now().Nanosecond())
	pout, err := os.Create(fname)
	if err != nil {
		log.Println("Unable to create temp file", fname, "for process stderr/stdout")
		p.attr.Files = []*os.File{nil, os.Stdout, os.Stderr}
	}

	// var procAttr os.ProcAttr
	// procAttr.Files = []*os.File{nil, os.Stdout, os.Stderr}
	p.attr.Files = []*os.File{nil, pout, pout}

	p.Output = fname

	process, err := os.StartProcess(command[0], command, &p.attr)
	if err != nil {
		return nil, err
	}

	// detach process
	// process.Release()

	p.Process = process

	fmt.Println("Started", command, "All output goes to", fname)

	go func() {
		pid := p.Pid
		s, err := process.Wait()
		if err != nil {
			panic(err)
		}

		for i := range h.Procs {
			if h.Procs[i].Process == nil {
				continue
			}
			if h.Procs[i].Pid == pid {
				h.Procs[i].Pid = 0
			}
		}

		log.Printf("Process [%d] %v finished with %v, %v", pid, command, s.Exited(), s.String())
	}()

	return p, nil
}

// RunCommand prepares ip command to run
func (h Host) RunCommand(args ...string) (string, error) {
	var command []string

	if h.Cgroup != nil {
		command = h.Cgroup.CgExecCommand()
	}

	ipCmd := FullPathFor("ip")
	if ipCmd == "" {
		return "", errors.New("ip command not found the PATH")
	}

	if h.NetNs() != nil {
		command = append(command, []string{ipCmd, "netns", "exec", h.NetNs().Name()}...)
	}

	command = append(command, args...)

	return RunCommand(command[0], command[1:]...)
}

func (h Host) enableForwarding() error {
	_, err := h.RunCommand("sysctl", "net.ipv4.ip_forward=1")
	return err
}

// NodeName host name getter
func (h Host) NodeName() string {
	return h.Name
}

// NetNs getter
func (h Host) NetNs() *NetNs {
	return h.netns
}

// LinksCount getter
func (h Host) LinksCount() int {
	return len(h.Links)
}

// GetCidr getter
func (h Host) GetCidr(peer Peer) string {
	return h.Links.LinkByPeer(peer).Cidr
}

// GetHwAddr getter
func (h Host) GetHwAddr(peer Peer) string {
	return h.Links.LinkByPeer(peer).HwAddr
}

// GetState getter
func (h Host) GetState(peer Peer) string {
	return h.Links.LinkByPeer(peer).State
}

// GetLinks links getter
func (h Host) GetLinks() Links {
	return h.Links
}

// Release does clean up
func (h Host) Release() error {
	if err := h.netns.Release(); err != nil {
		log.Println(err)
	}

	for _, link := range h.Links {
		link.Release()
	}

	for _, proc := range h.Procs {
		proc.Stop()
	}

	h.Cgroup.Release()

	return nil
}

// AddLink add link into host's links array
func (h *Host) AddLink(l Link) error {
	h.Links = append(h.Links, l)
	return nil
}

func (h *Host) recoverProcs() error {
	for i, proc := range h.Procs {
		fmt.Println("Recovering ", proc.Command, proc.Args)

		var p *os.Process
		var err error

		if p, err = proc.findProcessByName(h.NetNs().Name()); err != nil {
			return err
		}

		if p != nil {
			proc.Process = p
		} else {
			c := append([]string{proc.Command}, proc.Args...)
			result, err := h.runProcess(c...)
			if err != nil {
				return err
			}

			h.Procs[i].Process = result.Process

			continue
		}
	}

	return nil
}
