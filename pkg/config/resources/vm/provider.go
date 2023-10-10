package vm

import (
	"fmt"
	"os/exec"
	"os/user"
	"runtime"

	libvirt "github.com/digitalocean/go-libvirt"
	socket "github.com/digitalocean/go-libvirt/socket/dialers"
	"github.com/google/uuid"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"libvirt.org/go/libvirtxml"
)

// RandomID is a provider for generating random IDs
type Provider struct {
	config          *VM
	log             logger.Logger
	libvirtSocket   string
	libvirtQemuPath string
}

func (p *Provider) Init(cfg htypes.Resource, l logger.Logger) error {
	c, ok := cfg.(*VM)
	if !ok {
		return fmt.Errorf("unable to initialize VM provider, resource is not of type VM")
	}

	p.config = c
	p.log = l

	// TODO: how to detect this on windows?
	// Using linux as the default value.
	p.libvirtSocket = "/var/run/libvirt/libvirt-sock"
	p.libvirtQemuPath = "/usr/bin/qemu-system-x86_64"
	if runtime.GOOS == "darwin" {
		usr, err := user.Current()
		if err != nil {
			return fmt.Errorf("could not get current user: %v", err)
		}

		p.libvirtSocket = fmt.Sprintf("/Users/%s/.cache/libvirt/libvirt-sock", usr.Username)
	}

	return nil
}

func (p *Provider) Create() error {
	emulator, err := exec.LookPath(fmt.Sprintf("qemu-system-%s", p.config.Config.Arch))
	if err != nil {
		return fmt.Errorf("could not find qemu emulator: %v", err)
	}

	domainConfig := &libvirtxml.Domain{
		Type: "qemu",
		Name: p.config.Name,
		UUID: uuid.New().String(),
		Memory: &libvirtxml.DomainMemory{
			Value: uint(p.config.Resources.Memory),
			Unit:  "MB",
		},
		VCPU: &libvirtxml.DomainVCPU{
			Value: uint(p.config.Resources.CPU),
		},
		MemoryBacking: &libvirtxml.DomainMemoryBacking{
			MemorySource: &libvirtxml.DomainMemorySource{
				Type: "memfd",
			},
			MemoryAccess: &libvirtxml.DomainMemoryAccess{
				Mode: "shared",
			},
		},
		OS: &libvirtxml.DomainOS{
			Type: &libvirtxml.DomainOSType{
				Arch:    p.config.Config.Arch,
				Machine: "pc",
				Type:    "hvm",
			},
			BootDevices: []libvirtxml.DomainBootDevice{
				{
					Dev: "hd",
				},
			},
		},
		Devices: &libvirtxml.DomainDeviceList{
			Emulator: emulator,
			Disks: []libvirtxml.DomainDisk{
				{
					Driver: &libvirtxml.DomainDiskDriver{
						Name: "qemu",
						Type: "qcow2",
					},
					Source: &libvirtxml.DomainDiskSource{
						File: &libvirtxml.DomainDiskSourceFile{
							File: p.config.Image,
						},
					},
					Target: &libvirtxml.DomainDiskTarget{
						Dev: "hda",
					},
					Device: "disk",
				},
			},
			Filesystems: []libvirtxml.DomainFilesystem{
				{
					AccessMode: "passthrough",
					Driver: &libvirtxml.DomainFilesystemDriver{
						Type:  "virtiofs",
						Queue: 1024,
					},
					Source: &libvirtxml.DomainFilesystemSource{
						Mount: &libvirtxml.DomainFilesystemSourceMount{
							Dir: "/Users/erik/Downloads",
						},
					},
					Target: &libvirtxml.DomainFilesystemTarget{
						Dir: "mount_tag",
					},
				},
			},
			Graphics: []libvirtxml.DomainGraphic{
				{
					VNC: &libvirtxml.DomainGraphicVNC{
						Port:      -1,
						AutoPort:  "yes",
						WebSocket: 8999,
					},
				},
			},
			// Interfaces: []libvirtxml.DomainInterface{
			// 	{
			// 		Model: &libvirtxml.DomainInterfaceModel{
			// 			Type: "virtio",
			// 		},
			// 		Source: &libvirtxml.DomainInterfaceSource{
			// 			Bridge: &libvirtxml.DomainInterfaceSourceBridge{
			// 				Bridge: "bridge0",
			// 			},
			// 		},
			// 	},
			// },
		},
	}

	xml, err := domainConfig.Marshal()
	if err != nil {
		return fmt.Errorf("could not marshall vm config: %v", err)
	}

	dialer := socket.NewLocal(socket.WithSocket(p.libvirtSocket))
	client := libvirt.NewWithDialer(dialer)

	if err := client.ConnectToURI("qemu:///system"); err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	domain, err := client.DomainCreateXML(xml, libvirt.DomainNone)
	if err != nil {
		return fmt.Errorf("failed to create domains: %v", err)
	}

	p.config.UUID = domain.UUID

	return nil
}

func (p *Provider) Destroy() error {
	// detect this...
	dialer := socket.NewLocal(socket.WithSocket(p.libvirtSocket))

	client := libvirt.NewWithDialer(dialer)

	if err := client.ConnectToURI("qemu:///session"); err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	if err := client.DomainDestroy(libvirt.Domain{
		Name: p.config.Name,
		UUID: p.config.UUID,
	}); err != nil {
		return fmt.Errorf("failed to destroy domain: %v", err)
	}

	return nil
}

func (p *Provider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *Provider) Refresh() error {
	return nil
}

func (p *Provider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.ID)

	return false, nil
}
