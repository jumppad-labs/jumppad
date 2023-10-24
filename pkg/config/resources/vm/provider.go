package vm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	libvirt "github.com/digitalocean/go-libvirt"
	socket "github.com/digitalocean/go-libvirt/socket/dialers"
	"github.com/google/uuid"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/kdomanski/iso9660"
	"libvirt.org/go/libvirtxml"
)

const alphabet = "bcdefghijklmnopqrstuvwxyz"

type Provider struct {
	config   *VM
	log      logger.Logger
	client   *libvirt.Libvirt
	emulator string
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
	emulator, err := exec.LookPath(fmt.Sprintf("qemu-system-%s", p.config.Config.Arch))
	if err != nil {
		return fmt.Errorf("could not find qemu emulator: %v", err)
	}

	p.emulator = emulator

	socketpath := "/var/run/libvirt/libvirt-sock"
	if runtime.GOOS == "darwin" {
		// usr, err := user.Current()
		// if err != nil {
		// 	return fmt.Errorf("could not get current user: %v", err)
		// }

		// Default path for libvirt on macos when installed via brew.
		// socketpath = fmt.Sprintf("/Users/%s/.cache/libvirt/libvirt-sock", usr.Username)
		socketpath = "/Users/erik/.cache/libvirt/libvirt-sock"
	}

	dialer := socket.NewLocal(socket.WithSocket(socketpath))
	p.client = libvirt.NewWithDialer(dialer)

	return nil
}

func (p *Provider) Create() error {
	p.log.Info("Creating Virtual Machine", "ref", p.config.ID)

	if len(p.config.Disks) > len(alphabet) {
		return fmt.Errorf("virtual machines can only have %d disks", len(alphabet))
	}

	// Generate the disks that are mounted into the VM.
	disks := []libvirtxml.DomainDisk{}

	if p.config.CloudInit != nil {
		// Generate cloud-init iso.
		if err := p.generateCloudInit(); err != nil {
			return fmt.Errorf("failed to generate cloud-init iso: %v", err)
		}

		disks = append(disks, libvirtxml.DomainDisk{
			Driver: &libvirtxml.DomainDiskDriver{
				Name: "qemu",
				Type: "raw",
			},
			Source: &libvirtxml.DomainDiskSource{
				File: &libvirtxml.DomainDiskSourceFile{
					File: fmt.Sprintf("/tmp/%s/cloudinit.iso", p.config.Name),
				},
			},
			Target: &libvirtxml.DomainDiskTarget{
				Dev: "hde",
				Bus: "sata",
			},
			Device: "cdrom",
		})
	}

	for index, disk := range p.config.Disks {
		disks = append(disks, libvirtxml.DomainDisk{
			Driver: &libvirtxml.DomainDiskDriver{
				Name: "qemu",
				Type: disk.Type,
			},
			Source: &libvirtxml.DomainDiskSource{
				File: &libvirtxml.DomainDiskSourceFile{
					File: disk.Source,
				},
			},
			Target: &libvirtxml.DomainDiskTarget{
				Dev: fmt.Sprintf("vd%c", alphabet[index]),
			},
			Device: "disk",
			Boot: &libvirtxml.DomainDeviceBoot{
				Order: uint(index + 1),
			},
		})
	}

	// Generate the volumes that are mounted into the VM.
	volumes := []libvirtxml.DomainFilesystem{}
	for index, volume := range p.config.Volumes {
		volumes = append(volumes, libvirtxml.DomainFilesystem{
			AccessMode: "passthrough",
			// Driver: &libvirtxml.DomainFilesystemDriver{
			// 	Type:  "virtiofs",
			// 	Queue: 1024,
			// },
			Source: &libvirtxml.DomainFilesystemSource{
				Mount: &libvirtxml.DomainFilesystemSourceMount{
					Dir: volume.Source,
				},
			},
			Target: &libvirtxml.DomainFilesystemTarget{
				Dir: fmt.Sprintf("volume_%d", index),
			},
		})
	}

	// Generate the network devices that forward ports to the host.
	ports := []string{"user,id=n1"}
	for _, port := range p.config.Ports {
		ports = append(ports, fmt.Sprintf("hostfwd=tcp::%s-:%s", port.Host, port.Local))
	}
	netdev := strings.Join(ports, ",")

	// Generate controllers so we can add devices such as network interfaces
	// Set up the main controllers.
	controllers := []libvirtxml.DomainController{
		{
			Type:  "usb",
			Model: "qemu-xhci",
			Alias: &libvirtxml.DomainAlias{
				Name: "usb",
			},
		},
		{
			Type:  "pci",
			Model: "pcie-root",
			Alias: &libvirtxml.DomainAlias{
				Name: "pcie.0",
			},
		},
	}

	// Generate a controller for each PCI slot.
	for i := 1; i <= 5; i++ {
		controllers = append(controllers, libvirtxml.DomainController{
			Type:  "pci",
			Model: "pcie-root-port",
			Alias: &libvirtxml.DomainAlias{
				Name: fmt.Sprintf("pci.%d", i),
			},
		})
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
		OS: &libvirtxml.DomainOS{
			Type: &libvirtxml.DomainOSType{
				Arch:    p.config.Config.Arch,
				Machine: "pc-q35-3.1",
				Type:    "hvm",
			},
		},
		// need to figure out the right incantation here to sync clock with host
		Clock: &libvirtxml.DomainClock{
			Offset: "localtime",
			Timer: []libvirtxml.DomainTimer{
				{
					Name:       "rtc",
					Track:      "wall",
					TickPolicy: "catchup",
				},
			},
		},
		Devices: &libvirtxml.DomainDeviceList{
			Emulator:    p.emulator,
			Disks:       disks,
			Filesystems: volumes,
			Graphics: []libvirtxml.DomainGraphic{
				{
					VNC: &libvirtxml.DomainGraphicVNC{
						Port:      -1,
						AutoPort:  "yes",
						WebSocket: p.config.VNC.Port,
					},
				},
			},
			Videos: []libvirtxml.DomainVideo{
				{
					Model: libvirtxml.DomainVideoModel{
						Type: "virtio",
						// VRam: uint(32768),
					},
				},
			},
			Controllers: controllers,
			// Serials: []libvirtxml.DomainSerial{
			// 	{
			// 		Source: &libvirtxml.DomainChardevSource{
			// 			Pty: &libvirtxml.DomainChardevSourcePty{
			// 				Path: "/dev/ttys000",
			// 			},
			// 		},
			// 		Target: &libvirtxml.DomainSerialTarget{
			// 			Type:  "system-serial",
			// 			Port:  &zero,
			// 			Model: &libvirtxml.DomainSerialTargetModel{Name: "pl011"},
			// 		},
			// 		Alias: &libvirtxml.DomainAlias{
			// 			Name: "serial0",
			// 		},
			// 	},
			// },
			// Consoles: []libvirtxml.DomainConsole{
			// 	{
			// 		TTY: "/dev/ttys000",
			// 		Source: &libvirtxml.DomainChardevSource{
			// 			Pty: &libvirtxml.DomainChardevSourcePty{
			// 				Path: "/dev/ttys000",
			// 			},
			// 		},
			// 		Target: &libvirtxml.DomainConsoleTarget{
			// 			Type: "serial",
			// 			Port: &zero,
			// 		},
			// 		Alias: &libvirtxml.DomainAlias{
			// 			Name: "serial0",
			// 		},
			// 	},
			// },
			Inputs: []libvirtxml.DomainInput{
				{
					Type: "tablet",
					Bus:  "usb",
					Alias: &libvirtxml.DomainAlias{
						Name: "input0",
					},
				},
				{
					Type: "keyboard",
					Bus:  "usb",
					Alias: &libvirtxml.DomainAlias{
						Name: "input1",
					},
				},
			},
			/*
							<interface type='ethernet' name='eth0'>
				​  <start mode='onboot'/>
				​  <mac address='aa:bb:cc:dd:ee:ff'/>
				​  <protocol family='ipv4'>
				​    <dhcp/>
				​  </protocol>
				​</interface>*/
			Interfaces: []libvirtxml.DomainInterface{
				{
					Model: &libvirtxml.DomainInterfaceModel{
						Type: "virtio",
					},
					Source: &libvirtxml.DomainInterfaceSource{
						User: &libvirtxml.DomainInterfaceSourceUser{
							Dev: "ens2",
						},
					},
				},
				// {
				// 	Model: &libvirtxml.DomainInterfaceModel{
				// 		Type: "virtio",
				// 	},
				// 	Source: &libvirtxml.DomainInterfaceSource{
				// 		Bridge: &libvirtxml.DomainInterfaceSourceBridge{
				// 			Bridge: "bridge0",
				// 		},
				// 	},
				// },
			},
		},
		QEMUCommandline: &libvirtxml.DomainQEMUCommandline{
			Args: []libvirtxml.DomainQEMUCommandlineArg{
				// {Value: "-netdev"},
				// {Value: "user,id=mynet0,net=192.168.76.0/24,dhcpstart=192.168.76.9"},
				// {Value: "-nic"},
				// {Value: "vmnet-bridged,ifname=en0"},
				{Value: "-netdev"},
				{Value: netdev},
				{Value: "-device"},
				{Value: "virtio-net-pci,netdev=n1,bus=pcie.0,addr=0x19"},
				// {Value: "-smbios"},
				// {Value: "type=1,serial=ds='nocloud;s=http://10.0.2.2:8000/"},
			},
		},
	}

	// Exceptions for macOS on apple silicon.
	if runtime.GOOS == "darwin" && p.config.Config.Arch == "aarch64" {
		domainConfig.CPU = &libvirtxml.DomainCPU{
			Mode:  "custom",
			Match: "exact",
			Model: &libvirtxml.DomainCPUModel{
				Value: "host",
			},
		}

		domainConfig.Type = "hvf"
		domainConfig.OS.Firmware = "efi"
		domainConfig.OS.Type.Machine = "virt"
	}

	xml, err := domainConfig.Marshal()
	if err != nil {
		return fmt.Errorf("could not marshall vm config: %v", err)
	}

	if err := p.client.ConnectToURI("qemu:///session"); err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	_, err = p.client.DomainCreateXML(xml, libvirt.DomainNone)
	if err != nil {
		return fmt.Errorf("failed to create domains: %v", err)
	}

	return nil
}

func (p *Provider) Destroy() error {
	p.log.Info("Destroying Virtual Machine", "ref", p.config.ID)

	if err := p.client.ConnectToURI("qemu:///session"); err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	domain, err := p.client.DomainLookupByName(p.config.Name)
	if err != nil {
		return fmt.Errorf("failed to lookup domain: %v", err)
	}

	if err := p.client.DomainShutdown(domain); err != nil {
		return fmt.Errorf("failed to shutdown domain: %v", err)
	}

	if err := p.client.DomainDestroy(domain); err != nil {
		return fmt.Errorf("failed to destroy domain: %v", err)
	}

	// Clean up cloud-init
	if err := os.RemoveAll(fmt.Sprintf("/tmp/%s.iso", p.config.Name)); err != nil {
		return fmt.Errorf("failed to remove cloud-init iso: %v", err)
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

// Generate cloud-init iso.
func (p *Provider) generateCloudInit() error {
	p.log.Debug("Generating cloud-init", "ref", p.config.ID)

	writer, err := iso9660.NewWriter()
	if err != nil {
		return fmt.Errorf("failed to create writer: %s", err)
	}
	defer writer.Cleanup()

	tmp := fmt.Sprintf("/tmp/%s", p.config.Name)
	err = os.MkdirAll(tmp, 0755)
	if err != nil {
		return fmt.Errorf("failed to create tmp directory: %s", err)
	}

	if p.config.CloudInit.NetworkConfig != "" {
		networkconfig := filepath.Join(tmp, "network-config")
		err = os.WriteFile(networkconfig, []byte(p.config.CloudInit.NetworkConfig), 0644)
		if err != nil {
			return fmt.Errorf("failed to write network-config file: %s", err)
		}

		nf, err := os.Open(networkconfig)
		if err != nil {
			return fmt.Errorf("failed to open network-config file: %s", err)
		}
		defer nf.Close()

		err = writer.AddFile(nf, "network-config")
		if err != nil {
			return fmt.Errorf("failed to add network-config file: %s", err)
		}
	}

	if p.config.CloudInit.MetaData != "" {
		metadata := filepath.Join(tmp, "meta-data")
		err = os.WriteFile(metadata, []byte(p.config.CloudInit.MetaData), 0644)
		if err != nil {
			return fmt.Errorf("failed to write meta-data file: %s", err)
		}

		mf, err := os.Open(metadata)
		if err != nil {
			return fmt.Errorf("failed to open meta-data file: %s", err)
		}
		defer mf.Close()

		err = writer.AddFile(mf, "meta-data")
		if err != nil {
			return fmt.Errorf("failed to add meta-data file: %s", err)
		}
	}

	if p.config.CloudInit.UserData != "" {
		userdata := filepath.Join(tmp, "user-data")
		err = os.WriteFile(userdata, []byte(p.config.CloudInit.UserData), 0644)
		if err != nil {
			return fmt.Errorf("failed to write user-data file: %s", err)
		}

		uf, err := os.Open(userdata)
		if err != nil {
			return fmt.Errorf("failed to open user-data file: %s", err)
		}
		defer uf.Close()

		err = writer.AddFile(uf, "user-data")
		if err != nil {
			return fmt.Errorf("failed to add user-data file: %s", err)
		}
	}

	outputFile, err := os.OpenFile(fmt.Sprintf("/tmp/%s/cloudinit.iso", p.config.Name), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to create iso file: %s", err)
	}

	err = writer.WriteTo(outputFile, "cidata")
	if err != nil {
		return fmt.Errorf("failed to write ISO image: %s", err)
	}

	err = outputFile.Close()
	if err != nil {
		return fmt.Errorf("failed to close iso file: %s", err)
	}

	return nil
}
