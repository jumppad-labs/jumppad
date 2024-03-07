package k8s

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"golang.org/x/xerrors"
	"helm.sh/helm/v3/pkg/kube"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// Kubernetes defines an interface for a Kuberenetes client
type Kubernetes interface {
	SetConfig(string) (Kubernetes, error)
	GetPods(string) (*v1.PodList, error)
	HealthCheckPods(ctx context.Context, selectors []string, timeout time.Duration) error
	Apply(files []string, waitUntilReady bool) error
	Delete(files []string) error
	GetPodLogs(ctx context.Context, podName, nameSpace string) (io.ReadCloser, error)
}

// KubernetesImpl is a concrete implementation of a Kubernetes client
type KubernetesImpl struct {
	clientset  *kubernetes.Clientset
	client     corev1.CoreV1Interface
	configPath string
	timeout    time.Duration
	l          logger.Logger
}

// NewKubernetes creates a new client for interacting with Kubernetes clusters
func NewKubernetes(t time.Duration, l logger.Logger) Kubernetes {
	return &KubernetesImpl{timeout: t, l: l}
}

// SetConfig for the Kubernetes cluster and clones the client
func (k *KubernetesImpl) SetConfig(kubeconfig string) (Kubernetes, error) {
	kc := NewKubernetes(k.timeout, k.l).(*KubernetesImpl)

	kc.configPath = kubeconfig
	st := time.Now()
	for {
		err := kc.setConfig()
		if err == nil {
			break
		}

		if time.Now().Sub(st) > kc.timeout {
			return nil, xerrors.Errorf("error waiting for kubernetes client, config: %s error: %w", kubeconfig, err)
		}

		// backoff
		time.Sleep(5 * time.Second)
	}

	return kc, nil
}

// setConfig retries setting the config and building the client APIs
// it is possible that the cluster is not fully ready when
// this operation is first called
func (k *KubernetesImpl) setConfig() error {
	config, err := clientcmd.BuildConfigFromFlags("", k.configPath)
	if err != nil {
		return err
	}

	// Set insecure as the k3s certs sometimes have missing ips
	// when using a remote Docker
	config.TLSClientConfig.Insecure = true
	config.TLSClientConfig.CAFile = ""
	config.TLSClientConfig.CAData = nil

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	k.clientset = clientset
	k.client = clientset.CoreV1()

	return nil
}

// GetPodLogs returns a io.ReadCloser,err for a given pods' logs
func (k *KubernetesImpl) GetPodLogs(ctx context.Context, podName, nameSpace string) (io.ReadCloser, error) {
	var plOpts v1.PodLogOptions
	return k.clientset.CoreV1().Pods(nameSpace).GetLogs(podName, &plOpts).Stream(ctx)
}

// GetPods returns the Kubernetes pods based on the label selector
func (k *KubernetesImpl) GetPods(selector string) (*v1.PodList, error) {
	lo := metav1.ListOptions{
		LabelSelector: selector,
	}
	pl, err := k.client.Pods("").List(context.Background(), lo)
	if err != nil {
		return nil, err
	}

	return pl, nil
}

// Apply Kubernetes YAML files at path
// if waitUntilReady is true then the client will block until all resources have been created
func (k *KubernetesImpl) Apply(files []string, waitUntilReady bool) error {
	allFiles, err := buildFileList(files)
	if err != nil {
		return err
	}

	s := kube.GetConfig(k.configPath, "default", "default")
	kc := kube.New(s)

	// process the files
	for _, f := range allFiles {
		k.l.Debug("Applying Kubernetes config", "file", f)
		err := applyFile(f, waitUntilReady, kc)
		if err != nil {
			return err
		}
	}

	return nil
}

// Delete Kuberentes YAML files at path
func (k *KubernetesImpl) Delete(files []string) error {
	allFiles, err := buildFileList(files)
	if err != nil {
		return err
	}

	s := kube.GetConfig(k.configPath, "default", "default")
	kc := kube.New(s)

	// process the files
	for _, f := range allFiles {
		k.l.Debug("Removing Kubernetes config", "file", f)

		err := deleteFile(f, kc)
		if err != nil {
			return err
		}
	}

	return nil
}

// HealthCheckPods uses the given selector to check that all pods are started
// and running.
// selectors are checked sequentially
// pods = ["component=server,app=consul", "component=client,app=consul"]
func (k *KubernetesImpl) HealthCheckPods(ctx context.Context, selectors []string, timeout time.Duration) error {
	// check all pods are running
	for _, s := range selectors {
		k.l.Debug("Health checking pods", "selector", s)
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled")
		}

		err := k.healthCheckSingle(ctx, s, timeout)
		if err != nil {
			return err
		}
	}

	return nil
}

// healthCheckSingle checks for running containers with the given selector
func (k *KubernetesImpl) healthCheckSingle(ctx context.Context, selector string, timeout time.Duration) error {
	st := time.Now()
	for {
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled")
		}

		// backoff
		time.Sleep(2 * time.Second)

		if time.Now().Sub(st) > timeout {
			return fmt.Errorf("Timeout waiting for pods %s to start", selector)
		}

		// GetPods may return an error if the API server is not available
		pl, err := k.GetPods(selector)
		if err != nil {
			k.l.Debug("Error getting pods, will retry", "selector", selector, "error", err)
			continue
		}

		// there should be at least 1 pod
		if len(pl.Items) < 1 {
			k.l.Debug("Less than one item returned, will retry", "selector", selector)
			continue
		}

		allRunning := true
		for _, pod := range pl.Items {
			if pod.Status.Phase != "Running" {
				allRunning = false
				k.l.Debug("Pod not running", "pod", pod.Name, "namespace", pod.Namespace, "status", pod.Status.Phase)
				break
			}

			for _, s := range pod.Status.Conditions {
				// Check that the state of the pod is ready
				if s.Type == v1.PodReady && s.Status != v1.ConditionTrue {
					allRunning = false
					k.l.Debug("Pod not ready", "pod", pod.Name, "namespace", pod.Namespace, "type", s.Type, "value", s.Status)
				}
			}
		}

		if allRunning {
			k.l.Debug("Pods ready", "selector", selector)
			break
		}
	}

	return nil
}

func buildFileList(files []string) ([]string, error) {
	allFiles := make([]string, 0)

	for _, f := range files {
		// parse all of the config into a string
		fi, err := os.Stat(f)
		if err != nil {
			return nil, err
		}

		if fi.IsDir() {
			// add all the yaml files in the directory
			files, err := filepath.Glob(path.Join(f, "*.yaml"))
			if err != nil {
				return nil, err
			}
			allFiles = append(allFiles, files...)

			// add all the yml files in the directory
			files, err = filepath.Glob(path.Join(f, "*.yml"))
			if err != nil {
				return nil, err
			}
			allFiles = append(allFiles, files...)
		} else {
			allFiles = append(allFiles, f)
		}
	}

	return allFiles, nil
}

func applyFile(path string, waitUntilReady bool, kc *kube.Client) error {
	f, err := os.Open(path)
	if err != nil {
		return xerrors.Errorf("Unable to open file: %w", err)
	}
	defer f.Close()

	r, err := kc.Build(f, true)
	if err != nil {
		return xerrors.Errorf("Unable to build resources for file %s: %w", path, err)
	}

	_, err = kc.Create(r)
	if err != nil {
		return xerrors.Errorf("Unable to create resources for file %s: %w", path, err)
	}

	if waitUntilReady {
		return kc.WatchUntilReady(r, 30*time.Second)
	}

	return nil
}

func deleteFile(path string, kc *kube.Client) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r, err := kc.Build(f, false)
	if err != nil {
		return err
	}

	_, errs := kc.Delete(r)
	if errs != nil {
		//TODO need to handle this better
		return xerrors.Errorf("Error deleting configuration for file %s: %w", path, errs)
	}

	return nil
}
