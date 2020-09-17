package mn

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// Procs is set of Process instances
type Procs []*Process

// Process definition
type Process struct {
	*os.Process
	Command string
	Args    []string
	attr    os.ProcAttr
	Output  string
}

// GetByPid gets process by pid
func (ps Procs) GetByPid(pid int) *Process {
	for i := range ps {
		if ps[i].Process == nil {
			continue
		}
		if ps[i].Pid == pid {
			return ps[i]
		}
	}

	return nil
}

// GetPid gets process pid
func (p Process) GetPid() int {
	if p.Process != nil {
		return p.Pid
	}

	return 0
}

// Stop sends Interrupt signal to the process
func (p Process) Stop() error {
	if p.Process == nil {
		return fmt.Errorf("No such process: %s %s", p.Command, p.Args)
	}

	if err := p.Signal(os.Interrupt); err != nil {
		return err
	}

	return nil
}

// os.FindProcess() actually doesn't find it on posix systems, it just
// populates the struct and does SetFinalizer
// Idiomatic go way to find process by pid is:
//
// 	if e := syscall.Kill(pid, syscall.Signal(0)); e != nil {
// 		return nil, NewSyscallError("find process", e)
// 	}
//
// But we can't be sure, that found process actually the same
// so it looks, that finding it by name makes sense.
func (p Process) findProcessByName(netns string) (*os.Process, error) {
	out, err := RunCommand("ps", "-A", "-eo", "%p,%a")
	if err != nil {
		return nil, err
	}

	name := p.Command + " " + strings.Join(p.Args, " ")

	lines := strings.Split(out, "\n")

	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) != 2 {
			continue
		}
		if parts[1] != name {
			continue
		}

		pid, err := strconv.Atoi(strings.Trim(parts[0], " "))
		if err != nil {
			return nil, err
		}

		// check that process is in the right netns
		if netns == netnsByPid(pid) {
			return os.FindProcess(pid)
		}
	}

	return nil, nil
}

// Beware, the old versions of ip utility don't support 'identify' command
func netnsByPid(pid int) string {
	out, err := RunCommand("ip", "netns", "identify", strconv.Itoa(pid))
	if err != nil {
		log.Println(err)
		return ""
	}

	return strings.Trim(out, "\n")
}
