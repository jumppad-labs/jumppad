package clients

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

type Kubernetes interface {
	SetConfig(string) error
	GetPods() (*v1.PodList, error)
}

type KubernetesImpl struct {
	client corev1.CoreV1Interface
}

func NewKubernetes() Kubernetes {
	return &KubernetesImpl{}
}

func (k *KubernetesImpl) SetConfig(kubeconfig string) error {

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	k.client = clientset.CoreV1()

	return nil
}

func (k *KubernetesImpl) GetPods() (*v1.PodList, error) {
	lo := metav1.ListOptions{}
	pl, err := k.client.Pods("").List(lo)
	if err != nil {
		return nil, err
	}

	return pl, nil
}
