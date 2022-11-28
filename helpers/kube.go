package helpers

import (
	"context"
	"flag"
	"path/filepath"
	"sync"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type K8sClient struct {
	Clientset *kubernetes.Clientset
}

var (
	K8s  = &K8sClient{}
	Ctx  = context.Background()
	once sync.Once
)

func Init() (*K8sClient, error) {
	var err error
	once.Do(func() {

		clusterConfig, err := getClusterConfig()
		if err != nil {
			panic(err.Error())
		}

		// create the clientset
		clientset, err := kubernetes.NewForConfig(clusterConfig)
		if err != nil {
			panic(err.Error())
		}
		K8s.Clientset = clientset
	})
	return K8s, err
}

func getClusterConfig() (*rest.Config, error) {
	k8sInCluster := false
	if k8sInCluster {
		return getInClusterConfig()
	}
	return getOutOfClusterConfig()
}

func getInClusterConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return config, err
}

func getOutOfClusterConfig() (*rest.Config, error) {
	var kubeconfig string
	var defaultPath string
	kcFlag := flag.Lookup("kubeconfig")
	if home := homedir.HomeDir(); home != "" {
		defaultPath = filepath.Join(home, ".kube", "config")
		if kcFlag == nil {
			flag.StringVar(&kubeconfig, "kubeconfig", defaultPath, "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = kcFlag.Value.String()
		}
	} else {
		defaultPath = ""
		if kcFlag == nil {
			flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
		} else {
			kubeconfig = kcFlag.Value.String()
		}
	}
	flag.Parse()

	if kubeconfig == "" {
		kubeconfig = defaultPath
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (k8s *K8sClient) GetPodByName(name string) (*v1.Pod, error) {
	namespace := k8s.GetNamespace()

	pod, err := k8s.Clientset.CoreV1().Pods(namespace).Get(Ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return pod, nil
}

func (k8s *K8sClient) GetRunningPods() ([]v1.Pod, error) {
	namespace := k8s.GetNamespace()

	pods, err := k8s.Clientset.CoreV1().Pods(namespace).List(Ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// collect only pods that are running
	var runningPods []v1.Pod
	for _, pod := range pods.Items {
		if pod.Status.Phase == v1.PodRunning {
			runningPods = append(runningPods, pod)
		}
	}

	return runningPods, nil
}

func (k8s *K8sClient) GetNamespace() string {
	clientCfg, _ := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	namespace := clientCfg.Contexts[clientCfg.CurrentContext].Namespace

	if namespace == "" {
		namespace = "default"
	}
	return namespace
}

func (k8s *K8sClient) GetNodes() ([]v1.Node, error) {
	nodes, err := k8s.Clientset.CoreV1().Nodes().List(Ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return nodes.Items, nil
}

func (k8s *K8sClient) GetNodeOfPod(pod v1.Pod) (*v1.Node, error) {
	node, err := k8s.Clientset.CoreV1().Nodes().Get(Ctx, pod.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return node, nil
}
