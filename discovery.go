package main

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
)

// InitializeInternalClient InitializeKubernetesClient initializes a Kubernetes client from the environment
func InitializeInternalClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

// InitializeKubernetesClientForExternalUse initializes a Kubernetes client from a KubeConfig
func InitializeKubernetesClientForExternalUse() (*kubernetes.Clientset, error) {
	kubeconfig := os.Getenv("KUBECONFIG")

	if kubeconfig == "" {
		kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

type NodeResourceInfo struct {
	NodeName string
	CPU      float64 // CPUs in the node
	Memory   float64 // Memory in GiB
	MaxPods  int     // Max Pods
}

func FetchClusterDataTwo(clientset *kubernetes.Clientset) ([]NodeResourceInfo, error) {
	labelSelector := "node-role.kubernetes.io/control-plane="
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	var nodesResourceInfo []NodeResourceInfo

	for _, node := range nodes.Items {
		cpu := node.Status.Allocatable.Cpu().MilliValue()
		memory := node.Status.Allocatable.Memory().Value()
		memoryInGiB := float64(memory) / (1024 * 1024 * 1024)

		maxPods := inferMaxPodsFromNode(node)

		nodesResourceInfo = append(nodesResourceInfo, NodeResourceInfo{
			NodeName: node.Name,
			CPU:      float64(cpu) / 1000, // Convert milliCPU to CPU
			Memory:   memoryInGiB,
			MaxPods:  maxPods,
		})
	}

	return nodesResourceInfo, nil
}

func inferMaxPodsFromNode(node corev1.Node) int {
	// Retrieve the allocatable pods from the node status
	if allocatablePods, exists := node.Status.Allocatable["pods"]; exists {
		// The value is a Quantity, which needs to be parsed to an int
		maxPods, b := allocatablePods.AsInt64()
		if b != true {
			// Return the defult if the value cannot be parsed or if scale didn't happen
			return 250
		}
		return int(maxPods)
	}

	// Default value if the allocatable pods are not set
	return 250
}
