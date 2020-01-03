package clients

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"helm.sh/helm/v3/pkg/kube"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// Kubernetes defines an interface for a Kuberenetes client
type Kubernetes interface {
	SetConfig(string) error
	GetPods(string) (*v1.PodList, error)
	Apply(files []string, waitUntilReady bool) error
	Delete(files []string) error
}

// KubernetesImpl is a concrete implementation of a Kubernetes client
type KubernetesImpl struct {
	clientset  *kubernetes.Clientset
	client     corev1.CoreV1Interface
	configPath string
}

// NewKubernetes creates a new client for interacting with Kubernetes clusters
func NewKubernetes() Kubernetes {
	return &KubernetesImpl{}
}

// SetConfig for the Kubernetes cluster
func (k *KubernetesImpl) SetConfig(kubeconfig string) error {
	k.configPath = kubeconfig

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	k.clientset = clientset
	k.client = clientset.CoreV1()

	return nil
}

// GetPods returns the Kubernetes pods based on the label selector
func (k *KubernetesImpl) GetPods(selector string) (*v1.PodList, error) {
	lo := metav1.ListOptions{
		LabelSelector: selector,
	}
	pl, err := k.client.Pods("").List(lo)
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
		applyFile(f, waitUntilReady, kc)
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
		deleteFile(f, kc)
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
		return err
	}
	defer f.Close()

	r, err := kc.Build(f, false)
	if err != nil {
		return err
	}

	_, err = kc.Create(r)
	if err != nil {
		return err
	}

	if waitUntilReady {
		kc.WatchUntilReady(r, 30*time.Second)
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
		return fmt.Errorf("Error deleting configuration: %v", errs)
	}

	return nil
}
