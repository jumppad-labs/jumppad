package container

import "github.com/jumppad-labs/jumppad/pkg/clients/container/types"

func (i Image) ToClientImage() types.Image {
	return types.Image{
		ID:       i.ID,
		Name:     i.Name,
		Username: i.Username,
		Password: i.Password,
	}
}

func (i Images) ToClientImages() []types.Image {
	imgs := []types.Image{}
	for _, im := range i {
		imgs = append(imgs, im.ToClientImage())
	}

	return imgs
}

func (n NetworkAttachment) ToClientNetworkAttachment() types.NetworkAttachment {
	return types.NetworkAttachment{
		ID:        n.ID,
		Name:      n.Name,
		IPAddress: n.IPAddress,
		Aliases:   n.Aliases,
	}
}

func (n NetworkAttachments) ToClientNetworkAttachments() []types.NetworkAttachment {
	nets := []types.NetworkAttachment{}
	for _, net := range n {
		nets = append(nets, net.ToClientNetworkAttachment())
	}

	return nets
}

func (v Volume) ToClientVolume() types.Volume {
	return types.Volume{
		Source:                      v.Source,
		Destination:                 v.Destination,
		Type:                        v.Type,
		ReadOnly:                    v.ReadOnly,
		BindPropagation:             v.BindPropagation,
		BindPropagationNonRecursive: v.BindPropagationNonRecursive,
	}
}

func (v Volumes) ToClientVolumes() []types.Volume {
	vols := []types.Volume{}
	for _, vol := range v {
		vols = append(vols, vol.ToClientVolume())
	}

	return vols
}

func (p Port) ToClientPort() types.Port {
	return types.Port{
		Local:         p.Local,
		Host:          p.Host,
		Remote:        p.Remote,
		Protocol:      p.Protocol,
		OpenInBrowser: p.OpenInBrowser,
	}
}

func (p Ports) ToClientPorts() []types.Port {
	ports := []types.Port{}
	for _, port := range p {
		ports = append(ports, port.ToClientPort())
	}

	return ports
}

func (p PortRange) ToClientPortRange() types.PortRange {
	return types.PortRange{
		Range:      p.Range,
		EnableHost: p.EnableHost,
		Protocol:   p.Protocol,
	}
}

func (p PortRanges) ToClientPortRanges() []types.PortRange {
	ports := []types.PortRange{}
	for _, port := range p {
		ports = append(ports, port.ToClientPortRange())
	}

	return ports
}
