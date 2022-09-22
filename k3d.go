package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// checks if local running cluster is a k3s cluster
func isK3d() (bool, error) {
	nodes, err := k8sClient.getNodes()
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

func rootIntoPod(podName string) error {
	pod, err := k8sClient.getPodByName(podName)
	if err != nil {
		return err
	}

	nodeOfPod, err := k8sClient.getNodeOfPod(*pod)
	if err != nil {
		return err
	}

	containerID := pod.Status.ContainerStatuses[0].ContainerID
	containerID = strings.TrimPrefix(containerID, "containerd://")
	runCCmd := fmt.Sprintf("runc --root /run/containerd/runc/k8s.io/ exec -t -u 0 %s sh", containerID)
	cmd := []string{"sh", "-c", runCCmd}

	err = execIntoDockerContainer(nodeOfPod.Name, cmd)
	if err != nil {
		return err
	}

	return nil
}

func execIntoDockerContainer(containerName string, cmds []string) error {

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
