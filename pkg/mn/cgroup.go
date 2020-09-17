package mn

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/3d0c/mininet/pkg/cgroup"
)

// Cgroup structure
type Cgroup struct {
	*cgroup.Cgroup
	Name        string
	Controllers []Controller
}

// Controller structure
type Controller struct {
	Name   string
	Params []Set
}

// Set is a basic key/value structure
type Set struct {
	Key   string
	Value interface{}
}

// NewCgroup initialise Cgroup instance and does following:
// 1. Init
// 2. Init Cgroup struct
// 3. Set controllers
// 4. Physically create cgroup
// 5. Set controllers values
func NewCgroup(name string) (*Cgroup, error) {
	this := &Cgroup{Name: name}

	cgroup.Init()

	this.Cgroup = cgroup.NewCgroup(name)

	return this, nil
}

// UnmarshalJSON satisfies Unarshller
func (c *Cgroup) UnmarshalJSON(b []byte) error {
	type tmp Cgroup
	cg := tmp{}

	cgroup.Init()

	if err := json.Unmarshal(b, &cg); err != nil {
		return err
	}

	c.Name = cg.Name
	c.Cgroup = cgroup.NewCgroup(cg.Name)

	if err := c.SetControllers(cg.Controllers); err != nil {
		return err
	}

	if err := c.Cgroup.Create(); err != nil {
		return err
	}

	if err := c.SetParams(cg.Controllers); err != nil {
		return err
	}

	c.Controllers = cg.Controllers

	return nil
}

// SetControllers add controller into Controllers collection
func (c *Cgroup) SetControllers(controllers []Controller) error {
	for _, controller := range controllers {
		_, err := c.AddController(controller.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

// SetParams sets controller parameters
func (c *Cgroup) SetParams(controllers []Controller) error {
	for _, controller := range controllers {
		ctrl := c.GetController(controller.Name)

		for _, cv := range controller.Params {
			switch t := cv.Value.(type) {
			case string:
				if err := ctrl.SetValueString(cv.Key, cv.Value.(string)); err != nil {
					return err
				}
			case float64:
				if err := ctrl.SetValueInt64(cv.Key, (int64)(cv.Value.(float64))); err != nil {
					return err
				}
			case bool:
				if err := ctrl.SetValueBool(cv.Key, cv.Value.(bool)); err != nil {
					return err
				}
			default:
				log.Println("Unexpected type:", t)
			}
		}
	}

	return nil
}

// Release recusively release cgroup
func (c *Cgroup) Release() {
	if c != nil {
		c.DeleteExt(cgroup.DeleteRecursive)
	}
}

// CgExecCommand prepares cgexec command
func (c *Cgroup) CgExecCommand() []string {
	var groups string

	command := []string{FullPathFor("cgexec"), "-g"}

	for _, controller := range c.Controllers {
		groups += c.Name + ","
	}

	groups = strings.TrimRight(groups, ",") + ":" + c.Name

	return append(command, groups)
}
