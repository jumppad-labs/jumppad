package clients

import (
	"os"
	"path"
	"path/filepath"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/helm/pkg/kube"
)

// Kubernetes defines an interface for a Kuberenetes client
type Kubernetes interface {
	SetConfig(string) error
	GetPods(string) (*v1.PodList, error)
	Apply(files []string, waitUntilReady bool) error
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
	allFiles := make([]string, 0)

	for _, f := range files {
		// parse all of the config into a string
		fi, err := os.Stat(f)
		if err != nil {
			return err
		}

		if fi.IsDir() {
			// add all the yaml files in the directory
			files, err := filepath.Glob(path.Join(f, "*.yaml"))
			if err != nil {
				return err
			}
			allFiles = append(allFiles, files...)

			// add all the yml files in the directory
			files, err := filepath.Glob(path.Join(f, "*.yml"))
			if err != nil {
				return err
			}
			allFiles = append(allFiles, files...)
		} else {
			allFiles = append(allFiles, f)
		}
	}

	s := kube.GetConfig(k.configPath, "default", "default")
	kc := kube.New(s)
	// process the files
	for _, f := range allFiles {
		f, err := os.Open(f)
		if err != nil {
			return err
		}
		defer f.Close()

		r, err := kc.Build(f, false)
		if err != nil {
			return err
		}

		err = kc.Create(r)
		if err != nil {
			return err
		}

		if waitUntilReady {
			kc.WatchUntilReady(r)
		}
	}

	return nil
}

/*
	abs, _ := filepath.Abs(files)

	yamlFiles, err := filepath.Glob(path.Join(abs, "*.yaml"))
	if err != nil {
		return err
	}

	ymlFiles, err := filepath.Glob(path.Join(abs, "*.yml"))
	if err != nil {
		return err
	}

	yamlFiles = append(yamlFiles, ymlFiles...)

	for _, f := range yamlFiles {
		err := k.applyFile(f)
		if err != nil {
			return xerrors.Errorf("unable to apply kubernetes file %s: %w", f, err)
		}
	}

	return nil
}

func (k *KubernetesImpl) applyFile(path string) error {
	d := yaml.NewYAMLOrJSONDecoder(f, 4096)
	dd := k.clientset.Discovery()
	apigroups, err := discovery.GetAPIGroupResources(dd)
	if err != nil {
		log.Fatal(err)
	}

	restmapper := discovery.NewRESTMapper(apigroups, meta.InterfacesForUnstructured)

	for {
		// https://github.com/kubernetes/apimachinery/blob/master/pkg/runtime/types.go
		ext := runtime.RawExtension{}
		if err := d.Decode(&ext); err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		fmt.Println("raw: ", string(ext.Raw))
		versions := &runtime.VersionedObjects{}
		//_, gvk, err := objectdecoder.Decode(ext.Raw,nil,versions)
		obj, gvk, err := unstructured.UnstructuredJSONScheme.Decode(ext.Raw, nil, versions)
		fmt.Println("obj: ", obj)

		// https://github.com/kubernetes/apimachinery/blob/master/pkg/api/meta/interfaces.go
		mapping, err := restmapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			log.Fatal(err)
		}

		restconfig := config
		restconfig.GroupVersion = &schema.GroupVersion{
			Group:   mapping.GroupVersionKind.Group,
			Version: mapping.GroupVersionKind.Version,
		}
		dclient, err := dynamic.NewClient(restconfig)
		if err != nil {
			log.Fatal(err)
		}

		// https://github.com/kubernetes/client-go/blob/master/discovery/discovery_client.go
		apiresourcelist, err := dd.ServerResources()
		if err != nil {
			log.Fatal(err)
		}
		var myapiresource metav1.APIResource
		for _, apiresourcegroup := range apiresourcelist {
			if apiresourcegroup.GroupVersion == mapping.GroupVersionKind.Version {
				for _, apiresource := range apiresourcegroup.APIResources {
					//fmt.Println(apiresource)

					if apiresource.Name == mapping.Resource && apiresource.Kind == mapping.GroupVersionKind.Kind {
						myapiresource = apiresource
					}
				}
			}
		}
		fmt.Println(myapiresource)
		// https://github.com/kubernetes/client-go/blob/master/dynamic/client.go

		var unstruct unstructured.Unstructured
		unstruct.Object = make(map[string]interface{})
		var blob interface{}
		if err := json.Unmarshal(ext.Raw, &blob); err != nil {
			log.Fatal(err)
		}
		unstruct.Object = blob.(map[string]interface{})
		fmt.Println("unstruct:", unstruct)
		ns := "default"
		if md, ok := unstruct.Object["metadata"]; ok {
			metadata := md.(map[string]interface{})
			if internalns, ok := metadata["namespace"]; ok {
				ns = internalns.(string)
			}
		}
		res := dclient.Resource(&myapiresource, ns)
		fmt.Println(res)
		us, err := res.Create(&unstruct)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("unstruct response:", us)

	}
}
*/
