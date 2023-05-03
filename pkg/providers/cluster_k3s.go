package providers

import (
	"bytes"
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
	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/shipyard-run/hclconfig/types"
	"golang.org/x/xerrors"
)

// https://github.com/rancher/k3d/blob/master/cli/commands.go

const k3sBaseImage = "shipyardrun/k3s"
const k3sBaseVersion = "v1.23.12"

var startTimeout = (300 * time.Second)

//var startTimeout = (60 * time.Second)

// K8sCluster defines a provider which can create Kubernetes clusters
type K8sCluster struct {
	config     *resources.K8sCluster
	client     clients.ContainerTasks
	kubeClient clients.Kubernetes
	httpClient clients.HTTP
	connector  clients.Connector
	log        hclog.Logger
}

// NewK8sCluster creates a new Kubernetes cluster provider
func NewK8sCluster(c *resources.K8sCluster, cc clients.ContainerTasks, kc clients.Kubernetes, hc clients.HTTP, co clients.Connector, l hclog.Logger) *K8sCluster {
	return &K8sCluster{c, cc, kc, hc, co, l}
}

// Create implements interface method to create a cluster of the specified type
func (c *K8sCluster) Create() error {
	switch c.config.Driver {
	case "k3s":
		return c.createK3s()
	default:
		return ErrClusterDriverNotImplemented
	}
}

// Destroy implements interface method to destroy a cluster
func (c *K8sCluster) Destroy() error {
	switch c.config.Driver {
	case "k3s":
		return c.destroyK3s()
	default:
		return ErrClusterDriverNotImplemented
	}
}

// Lookup the a clusters current state
func (c *K8sCluster) Lookup() ([]string, error) {
	return c.client.FindContainerIDs(utils.FQDN(fmt.Sprintf("server.%s", c.config.Name), c.config.Module, c.config.Type))
}

func (c *K8sCluster) createK3s() error {
	// create a named log
	c.log = c.log.Named(c.config.Name)

	c.log.Info("Creating Cluster", "ref", c.config.ID)

	// check the cluster does not already exist
	ids, err := c.Lookup()
	if err != nil {
		return err
	}

	if ids != nil && len(ids) > 0 {
		return ErrClusterExists
	}

	if c.config.Version == "" {
		c.config.Version = k3sBaseVersion
	}

	// set the image
	image := fmt.Sprintf("%s:%s", k3sBaseImage, c.config.Version)

	// pull the container image
	err = c.client.PullImage(resources.Image{Name: image}, false)
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
	name := fmt.Sprintf("server.%s", c.config.Name)
	cc := &resources.Container{
		ResourceMetadata: types.ResourceMetadata{
			Name:   name,
			Type:   c.config.Type,
			Module: c.config.Module,
		},
	}

	cc.ParentConfig = c.config.Metadata().ParentConfig

	cc.Image = &resources.Image{Name: image}
	cc.Networks = c.config.Networks
	cc.Privileged = true // k3s must run Privlidged

	// set the volume mount for the images
	cc.Volumes = []resources.Volume{
		resources.Volume{
			Source:      volID,
			Destination: "/cache",
			Type:        "volume",
		},
	}

	// if there are any custom volumes to mount
	for _, v := range c.config.Volumes {
		cc.Volumes = append(cc.Volumes, v)
	}

	// Add any custom environment variables
	cc.Environment = map[string]string{}

	// set the environment variables for the K3S_KUBECONFIG_OUTPUT and K3S_CLUSTER_SECRET
	cc.Environment["K3S_KUBECONFIG_OUTPUT"] = "/output/kubeconfig.yaml"
	cc.Environment["K3S_CLUSTER_SECRET"] = "mysupersecret"

	// only add the variables for the cache when the kubernetes version is >= v1.18.16
	sv, err := semver.NewConstraint(">= v1.18.16")
	if err != nil {
		// Handle constraint not being parsable.
		return err
	}

	v, err := semver.NewVersion(c.config.Version)
	if err != nil {
		return fmt.Errorf("kubernetes version is not valid semantic version: %s", err)
	}

	if sv.Check(v) {
		// load the CA from a file
		ca, err := ioutil.ReadFile(filepath.Join(utils.CertsDir(""), "/root.cert"))
		if err != nil {
			return fmt.Errorf("unable to read root CA for proxy: %s", err)
		}

		cc.Environment["HTTP_PROXY"] = utils.HTTPProxyAddress()
		cc.Environment["HTTPS_PROXY"] = utils.HTTPSProxyAddress()
		cc.Environment["NO_PROXY"] = utils.ProxyBypass
		cc.Environment["PROXY_CA"] = string(ca)
	}

	// add any custom environment variables
	for k, v := range c.config.Environment {
		cc.Environment[k] = v
	}

	// set the API server port to a random number
	c.config.APIPort = rand.Intn(utils.MaxRandomPort-utils.MinRandomPort) + utils.MinRandomPort
	c.config.ConnectorPort = rand.Intn(utils.MaxRandomPort-utils.MinRandomPort) + utils.MinRandomPort

	// determine the snapshotter, if a storage driver other than overlay is used then
	// snapshotter must be set to native or the container will not start
	snapShotter := "native"

	if c.client.EngineInfo().StorageDriver == clients.StorageDriverOverlay || c.client.EngineInfo().StorageDriver == clients.StorageDriverOverlay2 {
		snapShotter = "overlayfs"
	}

	// create the server address
	FQDN := fmt.Sprintf("server.%s", utils.FQDN(c.config.Name, c.config.Module, c.config.Type))
	c.config.FQDN = FQDN

	// Set the default startup args
	// Also set netfilter settings to fix behaviour introduced in Linux Kernel 5.12
	// https://k3d.io/faq/faq/#solved-nodes-fail-to-start-or-get-stuck-in-notready-state-with-log-nf_conntrack_max-permission-denied
	args := []string{
		"server",
		fmt.Sprintf("--https-listen-port=%d", c.config.APIPort),
		"--kube-proxy-arg=conntrack-max-per-core=0",
		"--no-deploy=traefik",
		fmt.Sprintf("--snapshotter=%s", snapShotter),
		fmt.Sprintf("--tls-san=%s", FQDN),
	}

	// expose the API server and Connector ports
	cc.Ports = []resources.Port{
		resources.Port{
			Local:    fmt.Sprintf("%d", c.config.APIPort),
			Host:     fmt.Sprintf("%d", c.config.APIPort),
			Protocol: "tcp",
		},
		resources.Port{
			Local:    fmt.Sprintf("%d", c.config.ConnectorPort),
			Host:     fmt.Sprintf("%d", c.config.ConnectorPort),
			Protocol: "tcp",
		},
		resources.Port{
			Local:    fmt.Sprintf("%d", c.config.ConnectorPort+1),
			Host:     fmt.Sprintf("%d", c.config.ConnectorPort+1),
			Protocol: "tcp",
		},
	}

	cc.PortRanges = c.config.PortRanges
	cc.Ports = append(cc.Ports, c.config.Ports...)

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

	// get the assigned ip addresses for the container
	// and set that to the config
	dc := c.client.ListNetworks(id)
	for _, n := range dc {
		c.log.Info("network", "net", n)
		for i, net := range c.config.Networks {
			if net.ID == n.ID {
				// set the assigned address and name
				c.config.Networks[i].AssignedAddress = n.AssignedAddress
				c.config.Networks[i].Name = n.Name
			}
		}
	}

	// set the external IP
	c.config.ExternalIP = utils.GetDockerIP()

	// get the Kubernetes config file and drop it in a temp folder
	kc, err := c.copyKubeConfig(id)
	if err != nil {
		return xerrors.Errorf("unable to copy Kubernetes config: %w", err)
	}

	// replace the server location in the kubeconfig file
	// and write to $HOME/.shipyard/config/[clustername]/kubeconfig.yml
	// we need to do this as Shipyard might be using a remote Docker engine
	config, err := c.createLocalKubeConfig(kc)
	if err != nil {
		return xerrors.Errorf("unable to create local Kubernetes config: %w", err)
	}

	c.config.KubeConfig = config

	// wait for all the default pods like core DNS to start running
	// before progressing
	// we might also need to wait for the api services to become ready
	// this could be done with the folowing command kubectl get apiservice
	c.kubeClient, err = c.kubeClient.SetConfig(config)
	if err != nil {
		return err
	}

	// ensure essential pods have started before announcing the resource is available
	err = c.kubeClient.HealthCheckPods([]string{"app=local-path-provisioner", "k8s-app=kube-dns"}, startTimeout)
	if err != nil {
		// fetch the logs from the container before exit
		lr, lerr := c.client.ContainerLogs(id, true, true)
		if lerr != nil {
			c.log.Error("unable to get logs from container", "error", lerr)
		}

		// copy the logs to the output
		io.Copy(c.log.StandardWriter(&hclog.StandardLoggerOptions{}), lr)

		return xerrors.Errorf("timeout waiting for Kubernetes default pods: %w", err)
	}

	// import the images to the servers container d instance
	// importing images means that k3s does not need to pull from a remote docker hub
	if c.config.Images != nil && len(c.config.Images) > 0 {
		err := c.ImportLocalDockerImages(utils.ImageVolumeName, id, c.config.Images, false)
		if err != nil {
			return xerrors.Errorf("unable to importing Docker images: %w", err)
		}
	}

	// start the connectorService
	c.log.Debug("Deploying connector")
	return c.deployConnector(c.config.ConnectorPort, c.config.ConnectorPort+1)
}

func (c *K8sCluster) waitForStart(id string) error {
	start := time.Now()

	for {
		// not running after timeout exceeded? Rollback and delete everything.
		if startTimeout != 0 && time.Now().After(start.Add(startTimeout)) {
			//deleteCluster()
			return errors.New("cluster creation exceeded specified timeout")
		}

		// scan container logs for a line that tells us that the required services are up and running
		out, err := c.client.ContainerLogs(id, true, true)
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

func (c *K8sCluster) copyKubeConfig(id string) (string, error) {
	// create destination kubeconfig file paths
	_, kubePath, _ := utils.CreateKubeConfigPath(c.config.Name)

	// get kubeconfig file from container and read contents
	err := c.client.CopyFromContainer(id, "/output/kubeconfig.yaml", kubePath)
	if err != nil {
		return "", err
	}

	return kubePath, nil
}

func (c *K8sCluster) createLocalKubeConfig(kubeconfig string) (string, error) {
	ip := utils.GetDockerIP()
	_, kubePath, _ := utils.CreateKubeConfigPath(c.config.Name)

	err := c.changeServerAddressInK8sConfig(
		fmt.Sprintf("https://%s", ip),
		kubeconfig,
		kubePath,
	)
	if err != nil {
		return "", err
	}

	return kubePath, nil
}

func (c *K8sCluster) changeServerAddressInK8sConfig(addr, origFile, newFile string) error {
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
func (c *K8sCluster) deployConnector(grpcPort, httpPort int) error {
	// generate the certificates for the service
	cb, err := c.connector.GetLocalCertBundle(utils.CertsDir(""))
	if err != nil {
		return fmt.Errorf("unable to fetch root certificates for ingress: %s", err)
	}

	// generate the leaf certificates ensuring that we add
	// the ip address for the docker hosts as this might not be local
	lf, err := c.connector.GenerateLeafCert(
		cb.RootKeyPath,
		cb.RootCertPath,
		[]string{
			"connector",
			fmt.Sprintf("%s:%d", utils.GetDockerIP(), grpcPort),
		},
		[]string{utils.GetDockerIP()},
		utils.CertsDir(c.config.Name),
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
	c.log.Debug("Writing namespace config", "file", files[0])
	err = writeConnectorNamespace(files[0])
	if err != nil {
		return fmt.Errorf("unable to create namespace for connector: %s", err)
	}

	files = append(files, path.Join(dir, "secret.yaml"))
	c.log.Debug("Writing secret config", "file", files[1])
	writeConnectorK8sSecret(files[1], lf.RootCertPath, lf.LeafKeyPath, lf.LeafCertPath)
	if err != nil {
		return fmt.Errorf("unable to create secret for connector: %s", err)
	}

	files = append(files, path.Join(dir, "rbac.yaml"))
	c.log.Debug("Writing RBAC config", "file", files[2])
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
	c.log.Debug("Writing deployment config", "file", files[3])
	writeConnectorDeployment(files[3], grpcPort, httpPort, ll)
	if err != nil {
		return fmt.Errorf("unable to create deployment for connector: %s", err)
	}

	// deploy the application config
	err = c.kubeClient.Apply(files, true)
	if err != nil {
		return fmt.Errorf("unable to apply configuration: %s", err)
	}

	// wait for it to start
	c.kubeClient.HealthCheckPods([]string{"app=connector"}, 60*time.Second)
	if err != nil {
		return fmt.Errorf("timeout waiting for connector to start: %s", err)
	}

	return nil
}

// ImportLocalDockerImages fetches Docker images stored on the local client and imports them into the cluster
func (c *K8sCluster) ImportLocalDockerImages(name string, id string, images []resources.Image, force bool) error {
	imgs := []string{}

	for _, i := range images {
		// do nothing when the image name is empty
		if i.Name == "" {
			continue
		}

		err := c.client.PullImage(i, false)
		if err != nil {
			return err
		}

		imgs = append(imgs, i.Name)
	}

	// import to volume
	vn := utils.FQDNVolumeName(name)
	imagesFile, err := c.client.CopyLocalDockerImagesToVolume(imgs, vn, force)
	if err != nil {
		return err
	}

	for _, i := range imagesFile {
		// execute the command to import the image
		// write any command output to the logger
		err = c.client.ExecuteCommand(id, []string{"ctr", "image", "import", i}, nil, "/", "", "", c.log.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Debug}))
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *K8sCluster) destroyK3s() error {
	c.log.Info("Destroy Cluster", "ref", c.config.Name)

	ids, err := c.Lookup()
	if err != nil {
		return err
	}

	for _, i := range ids {
		err := c.client.RemoveContainer(i, false)
		if err != nil {
			return err
		}
	}

	_, kubePath, _ := utils.CreateKubeConfigPath(c.config.Name)
	os.RemoveAll(kubePath)

	return nil
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

var connectorDeployment = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: connector
  namespace: shipyard

---
apiVersion: v1
kind: Service
metadata:
  name: connector
  namespace: shipyard
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
  namespace: shipyard
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
        image: shipyardrun/connector:v0.1.0
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
  namespace: shipyard
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
  namespace: shipyard
subjects:
  - kind: ServiceAccount
    name: connector
    namespace: shipyard
roleRef:
  kind: ClusterRole
  name: service-creator
  apiGroup: rbac.authorization.k8s.io
`

var connectorNamespace = `
apiVersion: v1
kind: Namespace
metadata:
  name: shipyard
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
  namespace: shipyard
`
