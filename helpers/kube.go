package helpers

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"path/filepath"
	"strings"
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

func (k8s *K8sClient) GetPodByNamespaceAndName(namespace, name string) (*v1.Pod, error) {
	pod, err := k8s.Clientset.CoreV1().Pods(namespace).Get(Ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return pod, nil
}

func (k8s *K8sClient) GetPodsByImage(imageName string, exactMatch bool) ([]*v1.Pod, error) {
	allPods, err := k8s.GetPodsByNamespace("", false)
	if err != nil {
		return nil, err
	}

	foundPods := []*v1.Pod{}
	for _, pod := range allPods {
		for _, container := range pod.Spec.Containers {
			if exactMatch {
				if container.Image == imageName {
					foundPods = append(foundPods, pod)
				}
			} else {
				if strings.Contains(container.Image, imageName) {
					foundPods = append(foundPods, pod)
				}
			}
		}
	}

	return foundPods, nil
}

func (k8s *K8sClient) GetPodByNameInCurrentNamespace(name string) (*v1.Pod, error) {
	namespace := k8s.GetNamespace()

	return k8s.GetPodByNamespaceAndName(namespace, name)
}

func (k8s *K8sClient) GetAllRunningPods() ([]*v1.Pod, error) {
	return k8s.GetPodsByNamespace(metav1.NamespaceAll, true)
}

func (k8s *K8sClient) GetAllPods() ([]*v1.Pod, error) {
	return k8s.GetPodsByNamespace(metav1.NamespaceAll, false)
}

func (k8s *K8sClient) GetPodsByNamespace(namespace string, requireRunning bool) ([]*v1.Pod, error) {
	pods, err := k8s.Clientset.CoreV1().Pods(namespace).List(Ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// collect only pods that are running
	var runningPods []*v1.Pod
	for x, pod := range pods.Items {
		if pod.Status.Phase == v1.PodRunning || !requireRunning {
			runningPods = append(runningPods, &pods.Items[x])
		}
	}

	return runningPods, nil
}

func (k8s *K8sClient) GetRunningPodsInCurrentNamespace() ([]*v1.Pod, error) {
	namespace := k8s.GetNamespace()

	return k8s.GetPodsByNamespace(namespace, true)
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

func (k8s *K8sClient) GetLogsOfPod(pod v1.Pod) (string, error) {
	podLogOpts := v1.PodLogOptions{}
	req := k8s.Clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	podLogs, err := req.Stream(Ctx)
	if err != nil {
		return "", errors.New("error in opening stream")
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", errors.New("error in copy information from podLogs to buf")
	}
	str := buf.String()

	return str, nil
}

func (k8s *K8sClient) DeletePod(pod v1.Pod) error {
	err := k8s.Clientset.CoreV1().Pods(pod.Namespace).Delete(Ctx, pod.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (k8s *K8sClient) GetPVC(namespace, claimName string) (*v1.PersistentVolumeClaim, error) {
	pvc, err := k8s.Clientset.CoreV1().PersistentVolumeClaims(namespace).Get(Ctx, claimName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pvc, nil
}

func (k8s *K8sClient) GetPV(volumeName string) (*v1.PersistentVolume, error) {
	pv, err := k8s.Clientset.CoreV1().PersistentVolumes().Get(Ctx, volumeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pv, nil
}

func (k8s *K8sClient) GetHostPathOfVolumeMount(namespace string, volumeMount v1.Volume) (string, error) {
	pvc, err := k8s.GetPVC(namespace, volumeMount.PersistentVolumeClaim.ClaimName)
	if err != nil {
		return "", err
	}
	if pvc.Spec.VolumeName == "" {
		return "", errors.New("volumeName is empty")
	}
	pv, err := k8s.GetPV(pvc.Spec.VolumeName)
	if err != nil {
		return "", err
	}
	if pv.Spec.HostPath == nil {
		return "", errors.New("pv is not a hostpath")
	}
	return pv.Spec.HostPath.Path, nil
}
