package providers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/shipyard-run/shipyard/pkg/config"
	"golang.org/x/xerrors"
)

var (
	ErrorClusterInvalidName = errors.New("invalid cluster name")
)

// https://github.com/rancher/k3d/blob/master/cli/commands.go

const k3sBaseImage = "rancher/k3s"

func (c *Cluster) createK3s() error {
	c.log.Info("Creating Cluster", "ref", c.config.Name)

	// check the cluster name is valid
	if err := validateClusterName(c.config.Name); err != nil {
		return err
	}

	// check the cluster does not already exist
	id, err := c.Lookup()
	if id != "" {
		return ErrorClusterExists
	}

	// create the volume for the cluster
	volID, err := c.createVolume()
	if err != nil {
		return err
	}

	// set the image
	image := fmt.Sprintf("%s:%s", k3sBaseImage, c.config.Version)

	// create the server
	// since the server is just a container create the container config and provider
	cc := &config.Container{}
	cc.Name = fmt.Sprintf("server.%s", c.config.Name)
	cc.Image = config.Image{Name: image}
	cc.NetworkRef = c.config.NetworkRef
	cc.Privileged = true // k3s must run Privlidged

	// set the volume mount for the images
	cc.Volumes = []config.Volume{
		config.Volume{
			Source:      volID,
			Destination: "/images",
			Type:        "volume",
		},
	}

	// set the environment variables for the K3S_KUBECONFIG_OUTPUT and K3S_CLUSTER_SECRET
	cc.Environment = []config.KV{
		config.KV{Key: "K3S_KUBECONFIG_OUTPUT", Value: "/output/kubeconfig.yaml"},
		config.KV{Key: "K3S_CLUSTER_SECRET", Value: "mysupersecret"}, // This should be random
	}

	// set the API server port to a random number 64000 - 65000
	apiPort := rand.Intn(1000) + 64000
	args := []string{"server", fmt.Sprintf("--https-listen-port=%d", apiPort)}

	// expose the API server port
	cc.Ports = []config.Port{
		config.Port{
			Local:    apiPort,
			Host:     apiPort,
			Protocol: "tcp",
		},
	}

	// disable the installation of traefik
	args = append(args, "--no-deploy=traefik")

	cc.Command = args

	cp := NewContainer(cc, c.client, c.log.With("parent_ref", c.config.Name))
	err = cp.Create()
	if err != nil {
		return err
	}

	// get the id
	id, err = c.Lookup()
	if err != nil {
		return err
	}

	// wait for the server to start
	err = c.waitForStart(id)
	if err != nil {
		return err
	}

	// get the Kubernetes config file and drop it in $HOME/.shipyard/config/[clustername]/kubeconfig.yml
	kubeconfig, err := c.copyKubeConfig(id)
	if err != nil {
		return xerrors.Errorf("Error copying Kubernetes config: %w", err)
	}

	// create the Docker container version of the Kubeconfig
	// the default KubeConfig has the server location https://localhost:port
	// to use this config inside a docker container we need to use the FQDN for the server
	err = c.createDockerKubeConfig(kubeconfig)
	if err != nil {
		return xerrors.Errorf("Error creating Docker Kubernetes config: %w", err)
	}

	// janky  sleep to wait for API server before creating client, will make better
	time.Sleep(10 * time.Second)
	err = c.kubeClient.SetConfig(kubeconfig)
	if err != nil {
		return xerrors.Errorf("Error creating Kubernetes client: %w", err)
	}

	// wait for all the default pods like core DNS to start running
	// before progressing
	err = healthCheckPods(c.kubeClient, []string{""}, 120*time.Second, c.log.With("ref", c.config.Name))
	if err != nil {
		return xerrors.Errorf("Error while waiting for Kubernetes default pods: %w", err)
	}

	// import the images to the servers container d instance
	// importing images means that k3s does not need to pull from a remote docker hub
	if c.config.Images != nil && len(c.config.Images) > 0 {
		imageFile, err := writeLocalDockerImageToVolume(c.client, c.config.Images, volID, c.log)
		if err != nil {
			return err
		}

		// import the image
		//ctr image import filename
		err = execCommand(c.client, id, []string{"ctr", "image", "import", imageFile})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Cluster) waitForStart(id string) error {
	start := time.Now()
	timeout := 120 * time.Second

	for {
		// not running after timeout exceeded? Rollback and delete everything.
		if timeout != 0 && time.Now().After(start.Add(timeout)) {
			//deleteCluster()
			return errors.New("Cluster creation exceeded specified timeout")
		}

		// scan container logs for a line that tells us that the required services are up and running
		out, err := c.client.ContainerLogs(context.Background(), id, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
		if err != nil {
			out.Close()
			return fmt.Errorf(" Couldn't get docker logs for %s\n%+v", id, err)
		}
		buf := new(bytes.Buffer)
		nRead, _ := buf.ReadFrom(out)
		out.Close()
		output := buf.String()
		if nRead > 0 && strings.Contains(string(output), "Running kubelet") {
			break
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}

func (c *Cluster) copyKubeConfig(id string) (string, error) {
	// get kubeconfig file from container and read contents
	reader, _, err := c.client.CopyFromContainer(context.Background(), id, "/output/kubeconfig.yaml")
	if err != nil {
		return "", fmt.Errorf(" Couldn't copy kubeconfig.yaml from server container %s\n%+v", id, err)
	}
	defer reader.Close()

	readBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf(" Couldn't read kubeconfig from container\n%+v", err)
	}

	// write to file, skipping the first 512 bytes which contain file metadata
	// and trimming any NULL characters
	trimBytes := bytes.Trim(readBytes[512:], "\x00")

	// create destination kubeconfig file
	destDir, destPath, _ := CreateKubeConfigPath(c.config.Name)

	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		return "", err
	}

	kubeconfigfile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf(" Couldn't create kubeconfig file %s\n%+v", destPath, err)
	}

	defer kubeconfigfile.Close()
	kubeconfigfile.Write(trimBytes)

	return destPath, nil
}

func (c *Cluster) createDockerKubeConfig(kubeconfig string) error {
	// read the config into a string
	f, err := os.OpenFile(kubeconfig, os.O_RDONLY, 0755)
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
		fmt.Sprintf("server: https://server.%s", FQDN(c.config.Name, c.config.NetworkRef)),
		-1,
	)

	destPath := strings.Replace(kubeconfig, ".yaml", "-docker.yaml", 1)

	kubeconfigfile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("Couldn't create kubeconfig file %s\n%+v", destPath, err)
	}

	defer kubeconfigfile.Close()
	kubeconfigfile.Write([]byte(newConfig))

	return nil
}

func (c *Cluster) destroyK3s() error {
	c.log.Info("Delete Cluster", "ref", c.config.Name)

	cc := &config.Container{}
	cc.Name = fmt.Sprintf("server.%s", c.config.Name)
	cc.NetworkRef = c.config.NetworkRef

	cp := NewContainer(cc, c.client, c.log.With("parent_ref", c.config.Name))
	return cp.Destroy()
}

const clusterNameMaxSize int = 35

func validateClusterName(name string) error {
	if err := validateHostname(name); err != nil {
		return err
	}

	if len(name) > clusterNameMaxSize {
		return xerrors.Errorf("cluster name is too long (%d > %d): %w", len(name), clusterNameMaxSize, ErrorClusterInvalidName)
	}

	return nil
}

// ValidateHostname ensures that a cluster name is also a valid host name according to RFC 1123.
func validateHostname(name string) error {
	if len(name) == 0 {
		return xerrors.Errorf("no name provided %w", ErrorClusterInvalidName)
	}

	if name[0] == '-' || name[len(name)-1] == '-' {
		return xerrors.Errorf("hostname [%s] must not start or end with - (dash): %w", name, ErrorClusterInvalidName)
	}

	for _, c := range name {
		switch {
		case '0' <= c && c <= '9':
		case 'a' <= c && c <= 'z':
		case 'A' <= c && c <= 'Z':
		case c == '-':
			break
		default:
			return xerrors.Errorf("hostname [%s] contains characters other than 'Aa-Zz', '0-9' or '-': %w", ErrorClusterInvalidName)

		}
	}

	return nil
}