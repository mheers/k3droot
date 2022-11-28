package helpers

import (
	"context"
	"flag"
	"path/filepath"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type k8sClient struct {
	clientset *kubernetes.Clientset
}

var K8sClient = &k8sClient{}

func init() {
	clientset, err := getK8sClient()
	if err != nil {
		panic(err.Error())
	}
	K8sClient.clientset = clientset
}

func getK8sClient() (*kubernetes.Clientset, error) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, err
}

func (k8s *k8sClient) GetPodByName(name string) (*v1.Pod, error) {
	namespace := k8s.GetNamespace()

	pod, err := k8s.clientset.CoreV1().Pods(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return pod, nil
}

func (k8s *k8sClient) GetRunningPods() ([]v1.Pod, error) {
	namespace := k8s.GetNamespace()

	pods, err := k8s.clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
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

func (k8s *k8sClient) GetNamespace() string {
	clientCfg, _ := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	namespace := clientCfg.Contexts[clientCfg.CurrentContext].Namespace

	if namespace == "" {
		namespace = "default"
	}
	return namespace
}

func (k8s *k8sClient) GetNodes() ([]v1.Node, error) {
	nodes, err := k8s.clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return nodes.Items, nil
}

func (k8s *k8sClient) GetNodeOfPod(pod v1.Pod) (*v1.Node, error) {
	node, err := k8s.clientset.CoreV1().Nodes().Get(context.TODO(), pod.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return node, nil
}
