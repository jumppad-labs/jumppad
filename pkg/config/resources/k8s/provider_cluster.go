package k8s

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/connector"
	cclient "github.com/jumppad-labs/jumppad/pkg/clients/container"
	ctypes "github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/http"
	"github.com/jumppad-labs/jumppad/pkg/clients/k8s"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	sdk "github.com/jumppad-labs/plugin-sdk"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

// https://github.com/rancher/k3d/blob/master/cli/commands.go

var _ sdk.Provider = &ClusterProvider{}

var startTimeout = (300 * time.Second)

//var startTimeout = (60 * time.Second)

// K8sCluster defines a provider which can create Kubernetes clusters
type ClusterProvider struct {
	config     *Cluster
	client     cclient.ContainerTasks
	kubeClient k8s.Kubernetes
	httpClient http.HTTP
	connector  connector.Connector
	log        logger.Logger
}

func (p *ClusterProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*Cluster)
	if !ok {
		return fmt.Errorf("unable to initialize Kubernetes cluster provider, resource is not of type K8sCluster")
	}

	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	p.config = c
	p.client = cli.ContainerTasks
	p.kubeClient = cli.Kubernetes
	p.httpClient = cli.HTTP
	p.connector = cli.Connector
	p.log = l

	return nil
}

// Create implements interface method to create a cluster of the specified type
func (p *ClusterProvider) Create(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping create, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	return p.createK3s(ctx)
}

// Destroy implements interface method to destroy a cluster
func (p *ClusterProvider) Destroy(ctx context.Context, force bool) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping destroy, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	return p.destroyK3s(ctx, force)
}

// Lookup the a clusters current state
func (p *ClusterProvider) Lookup() ([]string, error) {
	return p.client.FindContainerIDs(utils.FQDN(fmt.Sprintf("server.%s", p.config.Meta.Name), p.config.Meta.Module, p.config.Meta.Type))
}

func (p *ClusterProvider) Refresh(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping refresh, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Debug("Refresh Kubernetes Cluster", "ref", p.config.Meta.Name)

	ci, err := p.getChangedImages()
	if err != nil {
		return err
	}

	if len(ci) > 0 {
		p.log.Info("Copied images changed, pushing new copy to the cluster", "ref", p.config.Meta.ID)
		err := p.ImportLocalDockerImages(ci, false)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *ClusterProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes Leaf Certificate", "ref", p.config.Meta.Name)

	// check to see if the any of the copied images have changed
	i, err := p.getChangedImages()
	if err != nil {
		return false, err
	}

	if len(i) > 0 {
		return true, nil
	}

	return false, nil
}

// ImportLocalDockerImages fetches Docker images stored on the local client and imports them into the cluster
func (p *ClusterProvider) ImportLocalDockerImages(images []ctypes.Image, force bool) error {
	id, err := p.Lookup()
	if err != nil {
		return err
	}

	imgs := []string{}

	for _, i := range images {
		// do nothing when the image name is empty
		if i.Name == "" {
			continue
		}

		err := p.client.PullImage(i, false)
		if err != nil {
			return err
		}

		imgs = append(imgs, i.Name)
	}

	// import to volume
	vn := utils.FQDNVolumeName(utils.ImageVolumeName)
	imagesFile, err := p.client.CopyLocalDockerImagesToVolume(imgs, vn, force)
	if err != nil {
		return err
	}

	for _, i := range imagesFile {
		p.log.Debug("Importing docker image", "ref", p.config.Meta.ID, "image", i)

		// execute the command to import the image
		// write any command output to the logger
		_, err = p.client.ExecuteCommand(id[0], []string{"ctr", "image", "import", i}, nil, "/", "", "", 300, p.log.StandardWriter())
		if err != nil {
			return err
		}
	}

	// prune the build images
	p.pruneBuildImages()

	// update the config with the image ids
	p.updateCopyImageIDs()

	return nil
}

func (p *ClusterProvider) pruneBuildImages() error {
	ids, err := p.Lookup()
	if err != nil {
		return err
	}

	// build a list of current images, we do not want to prune these
	filter := []string{}
	for _, i := range p.config.CopyImages {
		if strings.HasPrefix(i.Name, utils.BuildImagePrefix) {
			filter = append(filter, fmt.Sprintf("grep -v %s", i.Name))
		}
	}

	filters := strings.Join(filter, "| ")
	filters = strings.TrimSuffix(filters, "| ")

	command := fmt.Sprintf("ctr image rm $(ctr images ls name~=jumppad.dev/localcache/* -q | %s)", filters)

	p.log.Debug("Prune build images from nomad node", "id", ids[0], "command", command)
	output := bytes.NewBufferString("")
	_, _ = p.client.ExecuteCommand(ids[0], []string{"sh", "-c", command}, nil, "", "", "", 30, output)

	p.log.Debug("output", "result", output.String())
	return nil
}

func (p *ClusterProvider) updateCopyImageIDs() error {
	for n, i := range p.config.CopyImages {
		id, err := p.client.FindImageInLocalRegistry(i.ToClientImage())
		if err != nil {
			return err
		}

		p.config.CopyImages[n].ID = id
	}

	return nil
}

func (p *ClusterProvider) getChangedImages() ([]ctypes.Image, error) {
	changed := []ctypes.Image{}

	for _, i := range p.config.CopyImages {
		// has the image id changed
		id, err := p.client.FindImageInLocalRegistry(i.ToClientImage())
		if err != nil {
			p.log.Error("Unable to lookup image in local registry", "ref", p.config.Meta.ID, "error", err)
			return nil, err
		}

		// check that the current registry id for the image is the same
		// as the image that was used to create this container
		if id != i.ID {
			p.log.Debug("Container image changed, needs refresh", "ref", p.config.Meta.Name, "image", i.Name)
			changed = append(changed, i.ToClientImage())
		}
	}

	return changed, nil
}

func (p *ClusterProvider) createK3s(ctx context.Context) error {
	p.log.Info("Creating Cluster", "ref", p.config.Meta.ID)

	// check the cluster does not already exist
	ids, err := p.Lookup()
	if err != nil {
		return err
	}

	if len(ids) > 0 {
		return fmt.Errorf("error, cluster exists")
	}

	img := ctypes.Image{Name: p.config.Image.Name, Username: p.config.Image.Username, Password: p.config.Image.Password}
	// pull the container image
	err = p.client.PullImage(img, false)
	if err != nil {
		return err
	}

	// create the volume for the cluster
	volID, err := p.client.CreateVolume("images")
	if err != nil {
		return err
	}

	// create the server
	name := fmt.Sprintf("server.%s", p.config.Meta.Name)
	fqrn := utils.FQDN(name, p.config.Meta.Module, p.config.Meta.Type)

	cc := &ctypes.Container{}
	cc.Name = fqrn

	cc.Image = &img
	cc.Privileged = true // k3s must run Privileged

	for _, v := range p.config.Networks {
		cc.Networks = append(cc.Networks, ctypes.NetworkAttachment{
			ID:        v.ID,
			Name:      v.Name,
			IPAddress: v.IPAddress,
			Aliases:   v.Aliases,
		})
	}

	// set the volume mount for the images
	cc.Volumes = []ctypes.Volume{
		ctypes.Volume{
			Source:      volID,
			Destination: "/cache",
			Type:        "volume",
		},
	}

	// if there are any custom volumes to mount
	for _, v := range p.config.Volumes {
		cc.Volumes = append(cc.Volumes, ctypes.Volume{
			Source:                      v.Source,
			Destination:                 v.Destination,
			Type:                        v.Type,
			ReadOnly:                    v.ReadOnly,
			BindPropagation:             v.BindPropagation,
			BindPropagationNonRecursive: v.BindPropagationNonRecursive,
			SelinuxRelabel:              v.SelinuxRelabel,
		})
	}

	// add the registries volume
	rc, err := p.createRegistriesConfig()
	if err != nil {
		return fmt.Errorf("unable to create registries.yaml: %s", err)
	}

	if rc != "" {
		cc.Volumes = append(cc.Volumes, ctypes.Volume{
			Source:      rc,
			Destination: "/etc/rancher/k3s/registries.yaml",
			Type:        "bind",
		})
	}

	// Add any custom environment variables
	cc.Environment = map[string]string{}

	// set the environment variables for the K3S_KUBECONFIG_OUTPUT and K3S_CLUSTER_SECRET
	cc.Environment["K3S_KUBECONFIG_OUTPUT"] = "/output/kubeconfig.yaml"

	// only add the variables for the cache when the kubernetes version is >= v1.18.16
	sv, err := semver.NewConstraint(">= v1.18.16")
	if err != nil {
		// Handle constraint not being parsable.
		return err
	}

	// get the version from the image so we can calculate parameters
	version := "v99"
	vParts := strings.Split(p.config.Image.Name, ":")
	if len(vParts) == 2 && vParts[1] != "latest" {
		version = vParts[1]
	}

	v, err := semver.NewVersion(version)
	if err != nil {
		return fmt.Errorf("kubernetes version is not valid semantic version: %s", err)
	}

	if sv.Check(v) {
		// load the CA from a file
		ca, err := os.ReadFile(filepath.Join(utils.CertsDir(""), "/root.cert"))
		if err != nil {
			return fmt.Errorf("unable to read root CA for proxy: %s", err)
		}

		cc.Environment["CONTAINERD_HTTP_PROXY"] = utils.ImageCacheAddress()
		cc.Environment["CONTAINERD_HTTPS_PROXY"] = utils.ImageCacheAddress()
		cc.Environment["PROXY_CA"] = string(ca)

		// add the no-proxy overrides
		if p.config.Config != nil &&
			p.config.Config.DockerConfig != nil &&
			len(p.config.Config.DockerConfig.NoProxy) > 0 {
			cc.Environment["CONTAINERD_NO_PROXY"] = strings.Join(p.config.Config.DockerConfig.NoProxy, ",")
		}
	}

	// add any custom environment variables
	for k, v := range p.config.Environment {
		cc.Environment[k] = v
	}

	// set the Connector server port to a random number
	p.config.ConnectorPort = rand.Intn(utils.MaxRandomPort-utils.MinRandomPort) + utils.MinRandomPort

	// determine the snapshotter, if a storage driver other than overlay is used then
	// snapshotter must be set to native or the container will not start
	snapShotter := "native"

	if p.client.EngineInfo().StorageDriver == ctypes.StorageDriverOverlay || p.client.EngineInfo().StorageDriver == ctypes.StorageDriverOverlay2 {
		snapShotter = "overlayfs"
	}

	// only add the variables for the cache when the kubernetes version is >= v1.18.16
	sv, err = semver.NewConstraint(">= v1.25.0")
	if err != nil {
		// Handle constraint not being parsable.
		return err
	}

	disableArgs := "--no-deploy=traefik"
	clusterToken := ""

	if sv.Check(v) {
		disableArgs = "--disable=traefik"
		clusterToken = "--token=mysupersecret"
	} else {
		// add the cluster secret as an env this is deprecated in v1.25 and
		// replaced with --token
		cc.Environment["K3S_CLUSTER_SECRET"] = "mysupersecret"
	}

	// create the server address
	FQDN := fmt.Sprintf("server.%s", utils.FQDN(p.config.Meta.Name, p.config.Meta.Module, p.config.Meta.Type))
	p.config.ContainerName = FQDN

	// Set the default startup args
	// Also set netfilter settings to fix behaviour introduced in Linux Kernel 5.12
	// https://k3d.io/faq/faq/#solved-nodes-fail-to-start-or-get-stuck-in-notready-state-with-log-nf_conntrack_max-permission-denied
	args := []string{
		"server",
		fmt.Sprintf("--https-listen-port=%d", p.config.APIPort),
		"--kube-proxy-arg=conntrack-max-per-core=0",
		disableArgs,
		fmt.Sprintf("--snapshotter=%s", snapShotter),
		fmt.Sprintf("--tls-san=%s", FQDN),
		clusterToken,
	}

	// expose the API server and Connector ports
	cc.Ports = []ctypes.Port{
		ctypes.Port{
			Local:    fmt.Sprintf("%d", p.config.APIPort),
			Host:     fmt.Sprintf("%d", p.config.APIPort),
			Protocol: "tcp",
		},
		ctypes.Port{
			Local:    fmt.Sprintf("%d", p.config.ConnectorPort),
			Host:     fmt.Sprintf("%d", p.config.ConnectorPort),
			Protocol: "tcp",
		},
		ctypes.Port{
			Local:    fmt.Sprintf("%d", p.config.ConnectorPort+1),
			Host:     fmt.Sprintf("%d", p.config.ConnectorPort+1),
			Protocol: "tcp",
		},
	}

	for _, pr := range p.config.PortRanges {
		cc.PortRanges = append(cc.PortRanges, ctypes.PortRange{
			Range:      pr.Range,
			EnableHost: pr.EnableHost,
			Protocol:   pr.Protocol,
		})
	}

	for _, p := range p.config.Ports {
		cc.Ports = append(cc.Ports, ctypes.Port{
			Local:         p.Local,
			Remote:        p.Remote,
			Host:          p.Host,
			Protocol:      p.Protocol,
			OpenInBrowser: p.OpenInBrowser,
		})
	}

	cc.Command = args

	id, err := p.client.CreateContainer(cc)
	if err != nil {
		return err
	}

	// wait for the server to start
	err = p.waitForStart(ctx, id)
	if err != nil {
		return err
	}

	// get the assigned ip addresses for the container
	// and set that to the config
	dc := p.client.ListNetworks(id)
	for _, n := range dc {
		for i, net := range p.config.Networks {
			if net.ID == n.ID {
				// set the assigned address and name
				p.config.Networks[i].IPAddress = n.IPAddress
				p.config.Networks[i].Name = n.Name
			}
		}
	}

	// set the external IP
	p.config.ExternalIP = utils.GetDockerIP()

	// get the Kubernetes config file and drop it in a temp folder
	kc, err := p.copyKubeConfig(id)
	if err != nil {
		return xerrors.Errorf("unable to copy Kubernetes config: %w", err)
	}

	// replace the server location in the kubeconfig file
	// and write to $HOME/.shipyard/config/[clustername]/kubeconfig.yml
	// we need to do this as Shipyard might be using a remote Docker engine
	config, err := p.createLocalKubeConfig(kc)
	if err != nil {
		return xerrors.Errorf("unable to create local Kubernetes config: %w", err)
	}

	p.config.KubeConfig.ConfigPath = config

	// parse the kubeconfig and get the details
	data, err := os.ReadFile(config)
	if err != nil {
		return xerrors.Errorf("unable to read Kubernetes config: %w", err)
	}

	cfg := &Configuration{}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return xerrors.Errorf("unable to unmarshal Kubernetes config: %w", err)
	}

	p.config.KubeConfig.CA = cfg.Clusters[0].Cluster.CertificateAuthorityData
	p.config.KubeConfig.ClientCertificate = cfg.Users[0].User.ClientCertificateData
	p.config.KubeConfig.ClientKey = cfg.Users[0].User.ClientKeyData

	// wait for all the default pods like core DNS to start running
	// before progressing
	// we might also need to wait for the api services to become ready
	// this could be done with the folowing command kubectl get apiservice
	p.kubeClient, err = p.kubeClient.SetConfig(config)
	if err != nil {
		return err
	}

	// ensure essential pods have started before announcing the resource is available
	err = p.kubeClient.HealthCheckPods(ctx, []string{"app=local-path-provisioner", "k8s-app=kube-dns"}, startTimeout)
	if err != nil {
		// fetch the logs from the container before exit
		lr, lerr := p.client.ContainerLogs(id, true, true)
		if lerr != nil {
			p.log.Error("unable to get logs from container", "error", lerr)
		}

		// copy the logs to the output
		io.Copy(p.log.StandardWriter(), lr)

		return xerrors.Errorf("timeout waiting for Kubernetes default pods: %w", err)
	}

	// import the images to the servers container d instance
	// importing images means that k3s does not need to pull from a remote docker hub
	if len(p.config.CopyImages) > 0 {
		imgs := []ctypes.Image{}
		for _, i := range p.config.CopyImages {
			imgs = append(imgs, ctypes.Image{
				Name:     i.Name,
				Username: i.Username,
				Password: i.Password,
			})

		}

		err := p.ImportLocalDockerImages(imgs, false)
		if err != nil {
			return xerrors.Errorf("unable to importing Docker images: %w", err)
		}
	}

	// start the connectorService
	p.log.Debug("Deploying connector")
	return p.deployConnector(ctx, p.config.ConnectorPort, p.config.ConnectorPort+1)
}

func (p *ClusterProvider) waitForStart(ctx context.Context, id string) error {
	start := time.Now()

	for {
		if ctx.Err() != nil {
			return xerrors.Errorf("context cancelled, the cluster may be in an incoplete state")
		}

		// not running after timeout exceeded? Rollback and delete everything.
		if startTimeout != 0 && time.Now().After(start.Add(startTimeout)) {
			//deleteCluster()
			return errors.New("cluster creation exceeded specified timeout")
		}

		// scan container logs for a line that tells us that the required services are up and running
		out, err := p.client.ContainerLogs(id, true, true)
		if err != nil {
			out.Close()
			return fmt.Errorf("unable to get docker logs for %s\n%+v", id, err)
		}

		// read from the log and check for Kublet running
		buf := new(bytes.Buffer)
		nRead, _ := buf.ReadFrom(out)
		out.Close()
		output := buf.String()
		if nRead > 0 && strings.Contains(string(output), "Running kubelet") {
			break
		}

		// wait and try again
		time.Sleep(1 * time.Second)
	}

	return nil
}

func (p *ClusterProvider) copyKubeConfig(id string) (string, error) {
	// create destination kubeconfig file paths
	_, kubePath, _ := utils.CreateKubeConfigPath(p.config.Meta.ID)

	// get kubeconfig file from container and read contents
	err := p.client.CopyFromContainer(id, "/output/kubeconfig.yaml", kubePath)
	if err != nil {
		return "", err
	}

	return kubePath, nil
}

func (p *ClusterProvider) createLocalKubeConfig(kubeconfig string) (string, error) {
	ip := utils.GetDockerIP()
	_, kubePath, _ := utils.CreateKubeConfigPath(p.config.Meta.ID)

	err := p.changeServerAddressInK8sConfig(
		fmt.Sprintf("https://%s", ip),
		kubeconfig,
		kubePath,
	)
	if err != nil {
		return "", err
	}

	return kubePath, nil
}

func (p *ClusterProvider) changeServerAddressInK8sConfig(addr, origFile, newFile string) error {
	// read the config into a string
	f, err := os.OpenFile(origFile, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	readBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("unable to read kubeconfig, %v", err)
	}

	// manipulate the file
	newConfig := strings.Replace(
		string(readBytes),
		"server: https://127.0.0.1",
		fmt.Sprintf("server: %s", addr),
		-1,
	)

	kubeconfigfile, err := os.Create(newFile)
	if err != nil {
		return fmt.Errorf("could not create kubeconfig file %s\n%+v", newFile, err)
	}

	defer kubeconfigfile.Close()
	kubeconfigfile.Write([]byte(newConfig))

	return nil
}

// deployConnector deploys the connector service to the cluster
// once it has started
func (p *ClusterProvider) deployConnector(ctx context.Context, grpcPort, httpPort int) error {
	// generate the certificates for the service
	cb, err := p.connector.GetLocalCertBundle(utils.CertsDir(""))
	if err != nil {
		return fmt.Errorf("unable to fetch root certificates for ingress: %s", err)
	}

	// generate the leaf certificates ensuring that we add
	// the ip address for the docker hosts as this might not be local
	lf, err := p.connector.GenerateLeafCert(
		cb.RootKeyPath,
		cb.RootCertPath,
		[]string{
			"connector",
			fmt.Sprintf("%s:%d", utils.GetDockerIP(), grpcPort),
		},
		[]string{utils.GetDockerIP()},
		utils.CertsDir(p.config.Meta.Name),
	)

	if err != nil {
		return fmt.Errorf("unable to generate leaf certificates for ingress: %s", err)
	}

	// create a temp directory to write config to
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return fmt.Errorf("unable to create temporary directory: %s", err)
	}

	defer os.RemoveAll(dir)

	files := []string{}

	files = append(files, path.Join(dir, "namespace.yaml"))
	p.log.Debug("Writing namespace config", "file", files[0])
	err = writeConnectorNamespace(files[0])
	if err != nil {
		return fmt.Errorf("unable to create namespace for connector: %s", err)
	}

	files = append(files, path.Join(dir, "secret.yaml"))
	p.log.Debug("Writing secret config", "file", files[1])
	writeConnectorK8sSecret(files[1], lf.RootCertPath, lf.LeafKeyPath, lf.LeafCertPath)
	if err != nil {
		return fmt.Errorf("unable to create secret for connector: %s", err)
	}

	files = append(files, path.Join(dir, "rbac.yaml"))
	p.log.Debug("Writing RBAC config", "file", files[2])
	writeConnectorRBAC(files[2])
	if err != nil {
		return fmt.Errorf("unable to create RBAC for connector: %s", err)
	}

	// get the log level from the environment variable
	ll := os.Getenv("LOG_LEVEL")
	if ll == "" {
		ll = "info"
	}

	files = append(files, path.Join(dir, "deployment.yaml"))
	p.log.Debug("Writing deployment config", "file", files[3])
	writeConnectorDeployment(files[3], grpcPort, httpPort, ll)
	if err != nil {
		return fmt.Errorf("unable to create deployment for connector: %s", err)
	}

	// deploy the application config
	err = p.kubeClient.Apply(files, true)
	if err != nil {
		return fmt.Errorf("unable to apply configuration: %s", err)
	}

	// wait for it to start
	p.kubeClient.HealthCheckPods(ctx, []string{"app=connector"}, 60*time.Second)
	if err != nil {
		return fmt.Errorf("timeout waiting for connector to start: %s", err)
	}

	return nil
}

func (p *ClusterProvider) destroyK3s(ctx context.Context, force bool) error {
	p.log.Info("Destroy Cluster", "ref", p.config.Meta.ID)

	ids, err := p.Lookup()
	if err != nil {
		return err
	}

	for _, i := range ids {
		err := p.client.RemoveContainer(i, force)
		if err != nil {
			return err
		}
	}

	configDir, _, _ := utils.CreateKubeConfigPath(p.config.Meta.ID)
	os.RemoveAll(configDir)

	return nil
}

// createRegistriesConfig creates the k3s mirrors config for the cluster
func (p *ClusterProvider) createRegistriesConfig() (string, error) {
	dir, _, _ := utils.CreateKubeConfigPath(p.config.Meta.ID)
	daemonConfigPath := path.Join(dir, "registries.yaml")

	// remove any existing files, fail silently
	os.RemoveAll(daemonConfigPath)

	// create the docker config
	dc := dockerConfig{
		Mirrors: map[string]dockerMirror{},
	}

	// if the config is nil, do nothing
	if p.config.Config == nil ||
		p.config.Config.DockerConfig == nil ||
		len(p.config.Config.DockerConfig.InsecureRegistries) < 1 {
		return "", nil
	}

	for _, ir := range p.config.Config.DockerConfig.InsecureRegistries {
		dc.Mirrors[ir] = dockerMirror{
			Endpoints: []string{fmt.Sprintf("http://%s", ir)},
		}
	}

	// write the config to a file
	data, err := yaml.Marshal(&dc)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(daemonConfigPath, data, os.ModePerm)

	return daemonConfigPath, err
}

func writeConnectorNamespace(path string) error {
	return ioutil.WriteFile(path, []byte(connectorNamespace), os.ModePerm)
}

// writeK8sSecret writes a Kubernetes secret yaml to a file
func writeConnectorK8sSecret(path, root, key, cert string) error {
	// load the key and base64 encode
	kd, err := ioutil.ReadFile(key)
	if err != nil {
		return err
	}

	kb := base64.StdEncoding.EncodeToString(kd)

	// load the cert and base64 encode
	cd, err := ioutil.ReadFile(cert)
	if err != nil {
		return err
	}

	cb := base64.StdEncoding.EncodeToString(cd)

	// load the root cert and base64 encode
	rd, err := ioutil.ReadFile(root)
	if err != nil {
		return err
	}

	rb := base64.StdEncoding.EncodeToString(rd)

	return ioutil.WriteFile(path, []byte(
		fmt.Sprintf(connectorSecret, rb, cb, kb),
	), os.ModePerm)
}

func writeConnectorDeployment(path string, grpc, http int, logLevel string) error {
	return ioutil.WriteFile(path, []byte(
		fmt.Sprintf(connectorDeployment, grpc, http, logLevel),
	), os.ModePerm)
}

func writeConnectorRBAC(path string) error {
	return ioutil.WriteFile(path, []byte(connectorRBAC), os.ModePerm)
}

type dockerConfig struct {
	Mirrors map[string]dockerMirror `yaml:"mirrors"`
}

type dockerMirror struct {
	Endpoints []string `yaml:"endpoint"`
}

type Configuration struct {
	Clusters []struct {
		Cluster struct {
			CertificateAuthorityData string `yaml:"certificate-authority-data"`
		} `yaml:"cluster"`
	} `yaml:"clusters"`
	Users []struct {
		Name string `yaml:"name"`
		User struct {
			ClientCertificateData string `yaml:"client-certificate-data"`
			ClientKeyData         string `yaml:"client-key-data"`
		} `yaml:"user"`
	} `yaml:"users"`
}

var connectorDeployment = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: connector
  namespace: jumppad

---
apiVersion: v1
kind: Service
metadata:
  name: connector
  namespace: jumppad
spec:
  type: NodePort
  selector:
    app: connector
  ports:
    - port: 60000
      nodePort: %d
      targetPort: 60000
      name: grpc
    - port: 60001
      nodePort: %d
      targetPort: 60001
      name: http

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: connector-deployment
  namespace: jumppad
  labels:
    app: connector
spec:
  replicas: 1
  selector:
    matchLabels:
      app: connector
  template:
    metadata:
      labels:
        app: connector
    spec:
      serviceAccountName: connector
      containers:
      - name: connector
        imagePullPolicy: IfNotPresent
        image: ghcr.io/jumppad-labs/connector:v0.4.0
        ports:
          - name: grpc
            containerPort: 60000
          - name: http
            containerPort: 60001
        command: ["/connector", "run"]
        args: [
          "--grpc-bind=:60000",
          "--http-bind=:60001",
					"--root-cert-path=/etc/connector/tls/root.crt",
					"--server-cert-path=/etc/connector/tls/tls.crt",
					"--server-key-path=/etc/connector/tls/tls.key",
					"--namespace=jumppad",
          "--log-level=%s",
          "--integration=kubernetes"
        ]
        volumeMounts:
          - mountPath: "/etc/connector/tls"
            name: connector-tls
            readOnly: true
      volumes:
      - name: connector-tls
        secret:
          secretName: connector-certs
`

var connectorRBAC = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: service-creator
  namespace: jumppad
rules:
- apiGroups: [""]
  resources: ["services", "endpoints", "pods"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

---
apiVersion: rbac.authorization.k8s.io/v1
# This cluster role binding allows anyone in the "manager" group to read secrets in any namespace.
kind: ClusterRoleBinding
metadata:
  name: service-creator-global
  namespace: jumppad
subjects:
  - kind: ServiceAccount
    name: connector
    namespace: jumppad
roleRef:
  kind: ClusterRole
  name: service-creator
  apiGroup: rbac.authorization.k8s.io
`

var connectorNamespace = `
apiVersion: v1
kind: Namespace
metadata:
  name: jumppad
`

var connectorSecret = `
apiVersion: v1
data:
  root.crt: %s
  tls.crt: %s
  tls.key: %s
kind: Secret
metadata:
  name: connector-certs
  namespace: jumppad
`
