package vm

import (
	"path/filepath"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
)

const TypeVM string = "vm"

type VM struct {
	types.ResourceBase `hcl:",remain"`

	Kernel   string `hcl:"kernel" json:"kernel"`
	BootArgs string `hcl:"boot_args" json:"boot_args"`
	Initrd   string `hcl:"initrd" json:"initrd"`

	Disks []Disk `hcl:"disk,block" json:"disk"`

	Serial string `hcl:"serial" json:"serial"`
}

type Disk struct {
	Path string `hcl:"path" json:"path"`
}

func (c *VM) Process() error {
	// use this firmware if no kernel is specified
	kernel, err := filepath.Abs(c.Kernel)
	if err != nil {
		return err
	}

	c.Kernel = kernel

	initrd, err := filepath.Abs(c.Initrd)
	if err != nil {
		return err
	}

	c.Initrd = initrd

	for index, disk := range c.Disks {
		path, err := filepath.Abs(disk.Path)
		if err != nil {
			return err
		}

		c.Disks[index].Path = path
	}

	serial, err := filepath.Abs(c.Serial)
	if err != nil {
		return err
	}

	c.Serial = serial

	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.Meta.ID)
		if r != nil {
		}
	}

	return nil
}
