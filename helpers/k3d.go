package helpers

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	v1 "k8s.io/api/core/v1"
)

// checks if local running cluster is a k3s cluster
func IsK3d() (bool, error) {
	nodes, err := K8s.GetNodes()
	if err != nil {
		return false, err
	}

	for _, node := range nodes {
		if strings.HasPrefix(node.Name, "k3d") {
			return true, nil
		}
	}

	return false, nil
}

func ExecInNamespacePodContainer(namespace, podName, containerName string, cmd []string) error {
	pod, err := K8s.GetPodByNamespaceAndName(namespace, podName)
	if err != nil {
		return err
	}

	nodeOfPod, err := K8s.GetNodeOfPod(*pod)
	if err != nil {
		return err
	}

	containerID := ""
	for _, c := range pod.Status.ContainerStatuses {
		if c.Name == containerName {
			containerID = c.ContainerID
			break
		}

	}
	if containerID == "" {
		return fmt.Errorf("container %s not found in pod %s", containerName, podName)
	}

	containerID = strings.TrimPrefix(containerID, "containerd://")
	runCCmd := fmt.Sprintf("runc --root /run/containerd/runc/k8s.io/ exec -t -u 0 %s %s", containerID, strings.Join(cmd, " "))
	containerCmd := []string{"sh", "-c", runCCmd}

	err = ExecIntoDockerContainer(nodeOfPod.Name, containerCmd)
	if err != nil {
		return err
	}

	return nil
}

func RootIntoNamespacePodContainer(namespace, podName, containerName, shell string) error {
	cmd := []string{shell}
	return ExecInNamespacePodContainer(namespace, podName, containerName, cmd)
}

func RootIntoPodContainer(podContainerName string) error {
	seperator := ": "
	podName := strings.Split(podContainerName, seperator)[0]
	containerName := strings.Split(podContainerName, seperator)[1]
	namespace := K8s.GetNamespace()

	return RootIntoNamespacePodContainer(namespace, podName, containerName, "sh")
}

func RunInNodeOfPod(pod v1.Pod, cmds []string) error {
	nodeOfPod, err := K8s.GetNodeOfPod(pod)
	if err != nil {
		return err
	}
	return ExecIntoDockerContainer(nodeOfPod.Name, cmds)
}

func ExecIntoDockerContainer(containerName string, cmds []string) error {

	prg := "docker"

	arg1 := "exec"
	arg2 := "-ti"
	arg3 := containerName

	args := []string{arg1, arg2, arg3}
	args = append(args, cmds...)

	cmd := exec.Command(prg, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	return nil
}
