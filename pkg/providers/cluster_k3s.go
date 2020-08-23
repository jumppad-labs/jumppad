package providers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"golang.org/x/xerrors"
)

// https://github.com/rancher/k3d/blob/master/cli/commands.go

const k3sBaseImage = "rancher/k3s"

var startTimeout = (300 * time.Second)

// K8sCluster defines a provider which can create Kubernetes clusters
type K8sCluster struct {
	config     *config.K8sCluster
	client     clients.ContainerTasks
	kubeClient clients.Kubernetes
	httpClient clients.HTTP
	log        hclog.Logger
}

// NewK8sCluster creates a new Kubernetes cluster provider
func NewK8sCluster(c *config.K8sCluster, cc clients.ContainerTasks, kc clients.Kubernetes, hc clients.HTTP, l hclog.Logger) *K8sCluster {
	return &K8sCluster{c, cc, kc, hc, l}
}

// Create implements interface method to create a cluster of the specified type
func (c *K8sCluster) Create() error {
	switch c.config.Driver {
	case "k3s":
		return c.createK3s()
	default:
		return ErrorClusterDriverNotImplemented
	}
}

// Destroy implements interface method to destroy a cluster
func (c *K8sCluster) Destroy() error {
	switch c.config.Driver {
	case "k3s":
		return c.destroyK3s()
	default:
		return ErrorClusterDriverNotImplemented
	}
}

// Lookup the a clusters current state
func (c *K8sCluster) Lookup() ([]string, error) {
	return c.client.FindContainerIDs(fmt.Sprintf("server.%s", c.config.Name), c.config.Type)
}

func (c *K8sCluster) createK3s() error {
	c.log.Info("Creating Cluster", "ref", c.config.Name)

	// check the cluster does not already exist
	ids, err := c.client.FindContainerIDs(fmt.Sprintf("server.%s", c.config.Name), c.config.Type)
	if err != nil {
		return err
	}

	if ids != nil && len(ids) > 0 {
		return ErrorClusterExists
	}

	// set the image
	image := fmt.Sprintf("%s:%s", k3sBaseImage, c.config.Version)

	// pull the container image
	err = c.client.PullImage(config.Image{Name: image}, false)
	if err != nil {
		return err
	}

	// create the volume for the cluster
	volID, err := c.client.CreateVolume("images")
	if err != nil {
		return err
	}

	// create the server
	// since the server is just a container create the container config and provider
	cc := config.NewContainer(fmt.Sprintf("server.%s", c.config.Name))
	c.config.ResourceInfo.AddChild(cc)

	cc.Image = &config.Image{Name: image}
	cc.Networks = c.config.Networks
	cc.Privileged = true // k3s must run Privlidged

	// set the volume mount for the images
	cc.Volumes = []config.Volume{
		config.Volume{
			Source:      volID,
			Destination: "/images",
			Type:        "volume",
		},
	}

	// if there are any custom volumes to mount
	for _, v := range c.config.Volumes {
		cc.Volumes = append(cc.Volumes, v)
	}

	// set the environment variables for the K3S_KUBECONFIG_OUTPUT and K3S_CLUSTER_SECRET
	cc.Environment = []config.KV{
		config.KV{Key: "K3S_KUBECONFIG_OUTPUT", Value: "/output/kubeconfig.yaml"},
		config.KV{Key: "K3S_CLUSTER_SECRET", Value: "mysupersecret"}, // This should be random
	}

	// set the API server port to a random number 64000 - 65000
	apiPort := rand.Intn(1000) + 64000
	connectorPort := rand.Intn(1000) + 64000
	args := []string{"server", fmt.Sprintf("--https-listen-port=%d", apiPort)}

	// save the config
	clusterConfig := clients.ClusterConfig{Address: "localhost", APIPort: apiPort, ConnectorPort: connectorPort, NodeCount: 1}
	_, configPath := utils.CreateClusterConfigPath(c.config.Name)

	err = clusterConfig.Save(configPath)
	if err != nil {
		return xerrors.Errorf("Unable to generate Cluster config: %w", err)
	}

	// expose the API server and Connector ports
	cc.Ports = []config.Port{
		config.Port{
			Local:    fmt.Sprintf("%d", apiPort),
			Host:     fmt.Sprintf("%d", apiPort),
			Protocol: "tcp",
		},
		config.Port{
			Local:    "19090",
			Host:     fmt.Sprintf("%d", connectorPort),
			Protocol: "tcp",
		},
	}

	// disable the installation of traefik
	args = append(args, "--no-deploy=traefik")
	cc.Command = args

	id, err := c.client.CreateContainer(cc)
	if err != nil {
		return err
	}

	// wait for the server to start
	err = c.waitForStart(id)
	if err != nil {
		return err
	}

	// get the Kubernetes config file and drop it in $HOME/.shipyard/config/[clustername]/kubeconfig.yml
	kc, err := c.copyKubeConfig(id)
	if err != nil {
		return xerrors.Errorf("Error copying Kubernetes config: %w", err)
	}

	// create the Docker container version of the Kubeconfig
	// the default KubeConfig has the server location https://localhost:port
	// to use this config inside a docker container we need to use the FQDN for the server
	err = c.createDockerKubeConfig(kc)
	if err != nil {
		return xerrors.Errorf("Error creating Docker Kubernetes config: %w", err)
	}

	// wait for all the default pods like core DNS to start running
	// before progressing
	// we might also need to wait for the api services to become ready
	// this could be done with the folowing command kubectl get apiservice
	err = c.kubeClient.SetConfig(kc)
	if err != nil {
		return err
	}

	err = c.kubeClient.HealthCheckPods([]string{""}, startTimeout)
	if err != nil {
		// fetch the logs from the container before exit
		lr, err := c.client.ContainerLogs(id, true, true)
		if err != nil {
			c.log.Error("Unable to get logs from container", "error", err)
		}

		// copy the logs to the output
		io.Copy(c.log.StandardWriter(&hclog.StandardLoggerOptions{}), lr)

		return xerrors.Errorf("Error while waiting for Kubernetes default pods: %w", err)
	}

	// import the images to the servers container d instance
	// importing images means that k3s does not need to pull from a remote docker hub
	if c.config.Images != nil && len(c.config.Images) > 0 {
		err := c.ImportLocalDockerImages(utils.ImageVolumeName, id, c.config.Images, false)
		if err != nil {
			return xerrors.Errorf("Error importing Docker images: %w", err)
		}
	}

	return nil
}

func (c *K8sCluster) waitForStart(id string) error {
	start := time.Now()

	for {
		// not running after timeout exceeded? Rollback and delete everything.
		if startTimeout != 0 && time.Now().After(start.Add(startTimeout)) {
			//deleteCluster()
			return errors.New("Cluster creation exceeded specified timeout")
		}

		// scan container logs for a line that tells us that the required services are up and running
		out, err := c.client.ContainerLogs(id, true, true)
		if err != nil {
			out.Close()
			return fmt.Errorf(" Couldn't get docker logs for %s\n%+v", id, err)
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

func (c *K8sCluster) copyKubeConfig(id string) (string, error) {
	// create destination kubeconfig file paths
	_, destPath, _ := utils.CreateKubeConfigPath(c.config.Name)

	// get kubeconfig file from container and read contents
	err := c.client.CopyFromContainer(id, "/output/kubeconfig.yaml", destPath)
	if err != nil {
		return "", err
	}

	return destPath, nil
}

func (c *K8sCluster) createDockerKubeConfig(kubeconfig string) error {
	// read the config into a string
	f, err := os.OpenFile(kubeconfig, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	readBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("Couldn't read kubeconfig, %v", err)
	}

	// manipulate the file
	newConfig := strings.Replace(
		string(readBytes),
		"server: https://127.0.0.1",
		fmt.Sprintf("server: https://server.%s", utils.FQDN(c.config.Name, string(c.config.Type))),
		-1,
	)

	_, _, dockerPath := utils.CreateKubeConfigPath(c.config.Name)

	kubeconfigfile, err := os.Create(dockerPath)
	if err != nil {
		return fmt.Errorf("Couldn't create kubeconfig file %s\n%+v", dockerPath, err)
	}

	defer kubeconfigfile.Close()
	kubeconfigfile.Write([]byte(newConfig))

	return nil
}

// ImportLocalDockerImages fetches Docker images stored on the local client and imports them into the cluster
func (c *K8sCluster) ImportLocalDockerImages(name string, id string, images []config.Image, force bool) error {
	imgs := []string{}

	for _, i := range images {
		err := c.client.PullImage(i, false)
		if err != nil {
			return err
		}

		imgs = append(imgs, i.Name)
	}

	// import to volume
	vn := utils.FQDNVolumeName(name)
	imagesFile, err := c.client.CopyLocalDockerImageToVolume(imgs, vn, force)
	if err != nil {
		return err
	}

	for _, i := range imagesFile {
		// execute the command to import the image
		// write any command output to the logger
		err = c.client.ExecuteCommand(id, []string{"ctr", "image", "import", "/images/" + i}, nil, "/", c.log.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Debug}))
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *K8sCluster) destroyK3s() error {
	c.log.Info("Destroy Cluster", "ref", c.config.Name)

	ids, err := c.client.FindContainerIDs(fmt.Sprintf("server.%s", c.config.Name), c.config.Type)
	if err != nil {
		return err
	}

	for _, i := range ids {
		// remove from the networks
		for _, n := range c.config.Networks {
			err := c.client.DetachNetwork(n.Name, i)
			if err != nil {
				return err
			}
		}

		err := c.client.RemoveContainer(i)
		if err != nil {
			return err
		}
	}

	return nil
}
