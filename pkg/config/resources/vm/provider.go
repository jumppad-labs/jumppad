package vm

import (
	"context"
	"fmt"
	"log"

	htypes "github.com/jumppad-labs/hclconfig/types"
	sdk "github.com/jumppad-labs/plugin-sdk"

	hypervisor "github.com/jumppad-labs/cloudhypervisor-go-sdk"
	api "github.com/jumppad-labs/cloudhypervisor-go-sdk/api"
	"github.com/kr/pretty"
)

var _ sdk.Provider = &Provider{}

type Provider struct {
	config *VM
	log    sdk.Logger
}

func (p *Provider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*VM)
	if !ok {
		return fmt.Errorf("unable to initialize VM provider, resource is not of type VM")
	}

	p.config = c
	p.log = l

	return nil
}

func (p *Provider) Create(ctx context.Context) error {
	username := "jumppad"
	password := "$6$7125787751a8d18a$sHwGySomUA1PawiNFWVCKYQN.Ec.Wzz0JtPPL1MvzFrkwmop2dq7.4CYf03A5oemPQ4pOFCCrtCelvFBEle/K." // cloud123

	gateway := "192.168.249.1"
	cidr := "192.168.249.2/24"
	mac := "12:34:56:78:90:01"

	cloudinit, err := hypervisor.CreateCloudInitDisk("microvm", mac, cidr, gateway, username, password)
	if err != nil {
		return err
	}

	config := api.VmConfig{
		Payload: api.PayloadConfig{
			Kernel:    &p.config.Kernel,
			Initramfs: &p.config.Initrd,
			Cmdline:   &p.config.BootArgs,
		},
		Disks: &[]api.DiskConfig{
			{
				Path: cloudinit,
			},
		},
		Net: &[]api.NetConfig{
			{
				Mac: &mac,
			},
		},
		Cpus: &api.CpusConfig{
			BootVcpus: 1,
			MaxVcpus:  1,
		},
		Memory: &api.MemoryConfig{
			Size: 1024 * 1000 * 1000, // 1GB
		},
		Serial: &api.ConsoleConfig{
			Mode: "File",
			File: &p.config.Serial,
		},
	}

	disks := []api.DiskConfig{}
	for _, disk := range p.config.Disks {
		disks = append(disks, api.DiskConfig{
			Path: disk.Path,
		})
	}

	config.Disks = &disks

	pretty.Println(config)

	machine, err := hypervisor.NewMachine(ctx, config, log.New(p.log.StandardWriter(), "", log.LstdFlags))
	if err != nil {
		return err
	}

	err = machine.Start(ctx)
	if err != nil {
		return err
	}

	err = machine.Wait(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) Destroy(ctx context.Context, force bool) error {

	return nil
}

func (p *Provider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *Provider) Refresh(ctx context.Context) error {
	return nil
}

func (p *Provider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.Meta.ID)

	return false, nil
}
