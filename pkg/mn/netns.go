package mn

/*
#include <unistd.h>
#include <syscall.h>

int setns(int fd, int nstype) {
	return syscall(__NR_setns, fd, nstype);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
)

// NetnsRunDir default netns run dir
const NetnsRunDir = "/var/run/netns"

// NetNs definition
type NetNs struct {
	name string
}

// NewNetNs creates a NetNs instance
func NewNetNs(name string) (*NetNs, error) {
	netns := &NetNs{
		name: name,
	}

	if netns.Exists() {
		return this, nil
	}

	if err := netns.Create(); err != nil {
		return this, err
	}

	return this, nil
}

// Create network namespace
func (n NetNs) Create() error {
	if out, err := RunCommand("ip", "netns", "add", this.name); err != nil {
		return fmt.Errorf("Error: %v, output: %s", err, out)
	}

	return nil
}

// Exists check whether network namespace exists or not
func (n NetNs) Exists() bool {
	out, err := RunCommand("ip", "netns", "list")
	if err != nil {
		log.Printf("Error: %v, output: %s", err, out)
		return true
	}

	if strings.Contains(out, n.name) {
		return true
	}

	return false
}

// Release network namespace
func (n NetNs) Release() error {
	if n.name == "" {
		return nil
	}

	name := NetnsRunDir + "/" + n.name

	syscall.Unmount(name, syscall.MNT_DETACH)

	if err := os.Remove(name); err != nil {
		return err
	}

	return nil
}

// Name getter for network namespace
func (n NetNs) Name() string {
	return n.name
}
