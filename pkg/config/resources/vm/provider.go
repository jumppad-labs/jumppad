package vm

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	libvirt "github.com/digitalocean/go-libvirt"
	socket "github.com/digitalocean/go-libvirt/socket/dialers"
	"github.com/google/uuid"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/kdomanski/iso9660"
	"libvirt.org/go/libvirtxml"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

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
		usr, err := user.Current()
		if err != nil {
			return fmt.Errorf("could not get current user: %v", err)
		}

		// Default path for libvirt on macos when installed via brew.
		socketpath = fmt.Sprintf("/Users/%s/.cache/libvirt/libvirt-sock", usr.Username)
	}

	dialer := socket.NewLocal(socket.WithSocket(socketpath))
	p.client = libvirt.NewWithDialer(dialer)

	return nil
}

func (p *Provider) Create() error {
	p.log.Info("Creating Virtual Machine", "ref", p.config.ID)

	// Generate the disks that are mounted into the VM.
	disks := []libvirtxml.DomainDisk{}

	// Generate cloud-init iso if config was supplied.
	if p.config.CloudInit != nil {
		ci, err := p.createCloudInit()
		if err != nil {
			return fmt.Errorf("failed to generate cloud-init iso: %v", err)
		}

		disks = append(disks, ci)
	}

	// We only support 26 disks because we use the alphabet to name the disks.
	// Do we need to support more?
	if len(p.config.Disks) > len(alphabet) {
		return fmt.Errorf("virtual machines can only have %d disks", len(alphabet))
	}

	for index, disk := range p.config.Disks {
		dd, err := p.createDisk(disk.Type, disk.Source, index)
		if err != nil {
			return fmt.Errorf("failed to create disk: %v", err)
		}

		disks = append(disks, dd)
	}

	// Generate the volumes that are mounted into the VM.
	volumes := []libvirtxml.DomainFilesystem{}
	for index, volume := range p.config.Volumes {
		vo, err := p.createVolume(volume.Source, volume.Destination, index)
		if err != nil {
			return fmt.Errorf("failed to create volume: %v", err)
		}

		volumes = append(volumes, vo)
	}

	// Generate the network devices that forward ports to the host.
	// ports := []string{"user,id=n1"}
	// for _, port := range p.config.Ports {
	// 	ports = append(ports, fmt.Sprintf("hostfwd=tcp::%s-:%s", port.Host, port.Local))
	// }
	// netdev := strings.Join(ports, ",")

	// Generate controllers so we can add devices such as network interfaces
	// Set up the main controllers.
	usb, err := p.createController("usb", "qemu-xhci", "usb")
	if err != nil {
		return fmt.Errorf("failed to create usb controller: %v", err)
	}

	pcie, err := p.createController("pci", "pcie-root", "pcie.0")
	if err != nil {
		return fmt.Errorf("failed to create pcie controller: %v", err)
	}

	controllers := []libvirtxml.DomainController{
		usb,
		pcie,
	}

	// Generate a controller for each PCI slot.
	for i := 1; i <= 5; i++ {
		controller, err := p.createController("pci", "pcie-root-port", fmt.Sprintf("pcie.%d", i))
		if err != nil {
			return fmt.Errorf("failed to create controller: %v", err)
		}

		controllers = append(controllers, controller)
	}

	/*
		Generate the mac addresses for each network interface.
		Then pass those to network config and pre-generate the network config.
		Then create the interfaces that matches the network config.

		In the network config we can then "match" on the mac address.
	*/
	interfaces := []libvirtxml.DomainInterface{}
	for index := range p.config.Networks {
		nic, err := p.createNetworkInterface(index, "10.0.10.0/24")
		if err != nil {
			return fmt.Errorf("failed to create interface: %v", err)
		}

		p.log.Debug("Adding network interface", "ref", p.config.ID, "mac", nic.MAC.Address, "device", nic.Alias.Name)

		interfaces = append(interfaces, nic)
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
			// 			Type: "usb-serial",
			// 			// Port:  &zero,
			// 			Model: &libvirtxml.DomainSerialTargetModel{Name: "usb-serial"},
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
			// 			// Port: &zero,
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
			Interfaces: interfaces,
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
		QEMUCommandline: &libvirtxml.DomainQEMUCommandline{
			Args: []libvirtxml.DomainQEMUCommandlineArg{
				// {Value: "-net"},
				// {Value: "user,hostfwd=tcp::22222-:22"},
				// {Value: "-netdev"},
				// {Value: "user,id=mynet0,net=192.168.76.0/24,dhcpstart=192.168.76.9"},
				// {Value: "-nic"},
				// {Value: "vmnet-bridged,ifname=en0"},
				// {Value: "user,id=net0,net=10.0.10.0/24"},
				// {Value: netdev},
				//
				// WHY IS THIS NEEDED FOR CLOUD-INIT TO WORK?
				//
				{Value: "-netdev"},
				{Value: "user,id=net0"},
				{Value: "-device"},
				{Value: "virtio-net-pci,netdev=net0,bus=pcie.0,addr=0x19"},
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
	if err := os.RemoveAll(fmt.Sprintf("/tmp/%s", p.config.Name)); err != nil {
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

// Generate mac address.
func (p *Provider) generateMacAddress() string {
	buf := make([]byte, 6)
	_, err := rand.Read(buf)
	if err != nil {
		// If we couldn't generate random bytes, we have bigger problems.
		return "00:00:00:00:00:00"
	}
	buf[0] = (buf[0] | 2) & 0xfe // Set local bit, ensure unicast address
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
}

// Create Domain Disk
func (p *Provider) createDisk(diskType string, diskSource string, index int) (libvirtxml.DomainDisk, error) {
	disk := libvirtxml.DomainDisk{
		Driver: &libvirtxml.DomainDiskDriver{
			Name: "qemu",
			Type: diskType,
		},
		Source: &libvirtxml.DomainDiskSource{
			File: &libvirtxml.DomainDiskSourceFile{
				File: diskSource,
			},
		},
		Target: &libvirtxml.DomainDiskTarget{
			Dev: fmt.Sprintf("vd%c", alphabet[index]),
		},
		Device: "disk",
		Boot: &libvirtxml.DomainDeviceBoot{
			Order: uint(index + 1),
		},
	}

	return disk, nil
}

// Create volume
func (p *Provider) createVolume(source string, destination string, index int) (libvirtxml.DomainFilesystem, error) {
	volume := libvirtxml.DomainFilesystem{
		AccessMode: "passthrough",
		// Driver: &libvirtxml.DomainFilesystemDriver{
		// 	Type:  "virtiofs",
		// 	Queue: 1024,
		// },
		Source: &libvirtxml.DomainFilesystemSource{
			Mount: &libvirtxml.DomainFilesystemSourceMount{
				Dir: source,
			},
		},
		Target: &libvirtxml.DomainFilesystemTarget{
			Dir: fmt.Sprintf("volume_%d", index),
		},
	}

	return volume, nil
}

// Create controller
func (p *Provider) createController(controllerType string, controllerModel string, alias string) (libvirtxml.DomainController, error) {
	controller := libvirtxml.DomainController{
		Type:  controllerType,
		Model: controllerModel,
		Alias: &libvirtxml.DomainAlias{
			Name: alias,
		},
	}

	return controller, nil
}

// Create network interface
func (p *Provider) createNetworkInterface(index int, subnet string) (libvirtxml.DomainInterface, error) {
	mac := p.generateMacAddress()

	address, prefix, _ := strings.Cut(subnet, "/")

	mask, err := strconv.Atoi(prefix)
	if err != nil {
		return libvirtxml.DomainInterface{}, fmt.Errorf("failed to parse subnet prefix: %v", err)
	}

	nic := libvirtxml.DomainInterface{
		MAC: &libvirtxml.DomainInterfaceMAC{
			Address: mac,
		},
		IP: []libvirtxml.DomainInterfaceIP{
			{
				Family:  "ipv4",
				Address: address, // How to get this value from the network?
				Prefix:  uint(mask),
			},
		},
		// Source: &libvirtxml.DomainInterfaceSource{
		// 	User: &libvirtxml.DomainInterfaceSourceUser{
		// 		Dev: network.Device,
		// 	},
		// },
		Model: &libvirtxml.DomainInterfaceModel{
			Type: "virtio",
		},
		Alias: &libvirtxml.DomainAlias{
			Name: fmt.Sprintf("net%d", index),
		},
	}

	return nic, nil
}

// Create cloud-init iso.
func (p *Provider) createCloudInit() (libvirtxml.DomainDisk, error) {
	p.log.Debug("Generating cloud-init", "ref", p.config.ID)

	disk := libvirtxml.DomainDisk{}

	writer, err := iso9660.NewWriter()
	if err != nil {
		return disk, fmt.Errorf("failed to create writer: %s", err)
	}
	defer writer.Cleanup()

	tmp := fmt.Sprintf("/tmp/%s", p.config.Name)
	err = os.MkdirAll(tmp, 0755)
	if err != nil {
		return disk, fmt.Errorf("failed to create tmp directory: %s", err)
	}

	if p.config.CloudInit.NetworkConfig != "" {
		networkconfig := filepath.Join(tmp, "network-config")
		err = os.WriteFile(networkconfig, []byte(p.config.CloudInit.NetworkConfig), 0644)
		if err != nil {
			return disk, fmt.Errorf("failed to write network-config file: %s", err)
		}

		nf, err := os.Open(networkconfig)
		if err != nil {
			return disk, fmt.Errorf("failed to open network-config file: %s", err)
		}
		defer nf.Close()

		err = writer.AddFile(nf, "network-config")
		if err != nil {
			return disk, fmt.Errorf("failed to add network-config file: %s", err)
		}
	}

	if p.config.CloudInit.MetaData != "" {
		metadata := filepath.Join(tmp, "meta-data")
		err = os.WriteFile(metadata, []byte(p.config.CloudInit.MetaData), 0644)
		if err != nil {
			return disk, fmt.Errorf("failed to write meta-data file: %s", err)
		}

		mf, err := os.Open(metadata)
		if err != nil {
			return disk, fmt.Errorf("failed to open meta-data file: %s", err)
		}
		defer mf.Close()

		err = writer.AddFile(mf, "meta-data")
		if err != nil {
			return disk, fmt.Errorf("failed to add meta-data file: %s", err)
		}
	}

	if p.config.CloudInit.UserData != "" {
		userdata := filepath.Join(tmp, "user-data")
		err = os.WriteFile(userdata, []byte(p.config.CloudInit.UserData), 0644)
		if err != nil {
			return disk, fmt.Errorf("failed to write user-data file: %s", err)
		}

		uf, err := os.Open(userdata)
		if err != nil {
			return disk, fmt.Errorf("failed to open user-data file: %s", err)
		}
		defer uf.Close()

		err = writer.AddFile(uf, "user-data")
		if err != nil {
			return disk, fmt.Errorf("failed to add user-data file: %s", err)
		}
	}

	cloudinit := fmt.Sprintf("/tmp/%s/cloudinit.iso", p.config.Name)
	outputFile, err := os.OpenFile(cloudinit, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return disk, fmt.Errorf("failed to create iso file: %s", err)
	}

	err = writer.WriteTo(outputFile, "cidata")
	if err != nil {
		return disk, fmt.Errorf("failed to write ISO image: %s", err)
	}

	err = outputFile.Close()
	if err != nil {
		return disk, fmt.Errorf("failed to close iso file: %s", err)
	}

	disk = libvirtxml.DomainDisk{
		Driver: &libvirtxml.DomainDiskDriver{
			Name: "qemu",
			Type: "raw",
		},
		Source: &libvirtxml.DomainDiskSource{
			File: &libvirtxml.DomainDiskSourceFile{
				File: cloudinit,
			},
		},
		Target: &libvirtxml.DomainDiskTarget{
			Dev: "hda",
			Bus: "sata",
		},
		Device: "cdrom",
	}

	return disk, nil
}
