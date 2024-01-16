package main

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"math"
	"os"
	"strconv"
)

// Check https://access.redhat.com/documentation/en-us/red_hat_advanced_cluster_management_for_kubernetes/2.9/html/clusters/cluster_mce_overview#hosted-sizing-guidance
// for more details on the sizing guidance

// TODO Keep in mind that this is based on performance and scale regression fitting and is subject to change in the future as more testing
// is done and more data is collected
const (
	cpuRequestPerHCP    float64 = 5  // CPUs required per HCP
	memoryRequestPerHCP float64 = 18 // GB memory required per HCP
	podsPerHCP          float64 = 75 // Pods per HCP

	incrementalCPUUsagePer1KQPS float64 = 9.0 // Incremental CPU usage per 1K QPS
	incrementalMemUsagePer1KQPS float64 = 2.5 // Incremental Memory usage per 1K QPS

	idleCPUUsage    float64 = 2.9  // Idle CPU usage (unit vCPU)
	idleMemoryUsage float64 = 11.1 // Idle memory usage (unit GiB)
)

type ServerResources struct {
	WorkerCPUs        float64
	WorkerMemory      float64
	MaxPods           float64
	PodCount          float64
	APIRate           float64
	EtcdStorage       float64
	MaxHCPs           float64
	UseLoadBased      bool
	CalculationMethod int
}

// Linear regression constants for ETCD storage calculation derived from performance and scale regression fitting
const (
	etcdStorageSlope  float64 = 6.66e-4
	etcdStorageOffset float64 = 0.103
)

func calculateETCDStorage(podCount float64) float64 {
	return etcdStorageSlope*podCount + etcdStorageOffset
}

func calculateMaxHCPs(workerCPUs, workerMemory, maxPods, apiRate float64, useLoadBased bool) float64 {
	// print all values for debugging
	fmt.Printf("workerCPUs: %f\n", workerCPUs)
	fmt.Printf("workerMemory: %f\n", workerMemory)
	fmt.Printf("maxPods: %f\n", maxPods)
	fmt.Printf("apiRate: %f\n", apiRate)
	fmt.Printf("useLoadBased: %t\n", useLoadBased)

	maxHCPsByCPU := workerCPUs / cpuRequestPerHCP
	maxHCPsByMemory := workerMemory / memoryRequestPerHCP
	maxHCPsByPods := maxPods / podsPerHCP

	var maxHCPsByCPUUsage, maxHCPsByMemoryUsage float64
	if useLoadBased {
		maxHCPsByCPUUsage = workerCPUs / (idleCPUUsage + (apiRate/1000)*incrementalCPUUsagePer1KQPS)
		maxHCPsByMemoryUsage = workerMemory / (idleMemoryUsage + (apiRate/1000)*incrementalMemUsagePer1KQPS)
	} else {
		maxHCPsByCPUUsage = maxHCPsByCPU
		maxHCPsByMemoryUsage = maxHCPsByMemory
	}

	// Return the minimum of all the calculated values (this is the maximum number of HCPs that can be hosted)
	// This considers the most constrained resource as the limiting factor (e.g., CPU, Memory, Pods, etc.)
	minHCPs := math.Min(maxHCPsByCPU, math.Min(maxHCPsByCPUUsage, math.Min(maxHCPsByMemory, math.Min(maxHCPsByMemoryUsage, maxHCPsByPods))))

	return minHCPs
}

func promptForInput(promptLabel string) float64 {
	prompt := promptui.Prompt{
		Label: promptLabel,
		Validate: func(input string) error {
			if _, err := strconv.ParseFloat(input, 64); err != nil {
				return fmt.Errorf("invalid number")
			}
			return nil
		},
	}

	result, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}

	value, _ := strconv.ParseFloat(result, 64)
	return value
}

func promptForSelection(promptLabel string, items []string) int {
	prompt := promptui.Select{
		Label: promptLabel,
		Items: items,
	}

	_, result, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}

	for i, item := range items {
		if item == result {
			return i
		}
	}
	return -1
}

var rootCmd = &cobra.Command{
	Use:   "hcp-sizer",
	Short: "An HCP Sizing Calculator based on Science!",
	Run: func(cmd *cobra.Command, args []string) {
		resources := ServerResources{}

		resources.WorkerCPUs = promptForInput("Enter the number of vCPUs on the worker node")
		resources.WorkerMemory = promptForInput("Enter the memory (in GiB) on the worker node")
		resources.MaxPods = promptForInput("Enter the maximum number of pods on the worker node (usually 250 or 500)")
		resources.PodCount = promptForInput("Enter the number of pods you plan to run on your cluster (for ETCD storage calculation)")
		resources.CalculationMethod = promptForSelection("Select Calculation Method", []string{"Request-Based", "Load-Based"})
		resources.UseLoadBased = resources.CalculationMethod == 1

		// Check evaluation method, request-based or load-based (request is the more generic method)
		// load-based is preferred when data about QPS is available (e.g. from an existing cluster)

		if resources.UseLoadBased {
			green := color.New(color.FgGreen)

			italicGreen := green.Add(color.Italic)
			italicGreen.Println("❗️Hint: Run the following query in an existing cluster to estimate your QPS:")
			italicGreen.Println(`sum(rate(apiserver_request_total{namespace=~"clusters-$name*"}[2m])) by (namespace)`)

			resources.APIRate = promptForInput("Enter the estimated API rate (QPS)")
		}

		resources.MaxHCPs = calculateMaxHCPs(resources.WorkerCPUs, resources.WorkerMemory, resources.MaxPods, resources.APIRate, resources.UseLoadBased)
		resources.EtcdStorage = calculateETCDStorage(resources.PodCount)

		yellow := color.New(color.FgYellow)
		italitYellow := yellow.Add(color.Italic)

		// Print the results
		italitYellow.Printf("Maximum HCPs that can be hosted: %.2f\n", math.Floor(resources.MaxHCPs))
		italitYellow.Printf("Estimated HCP ETCD Storage Requirement: %.3f GiB\n", resources.EtcdStorage)
	},
}

func main() {
	//if err := rootCmd.Execute(); err != nil {
	//	fmt.Println(err)
	//	os.Exit(1)
	//}

	clientset, err := InitializeKubernetesClientForExternalUse()
	if err != nil {
		fmt.Println("Failed to initialize Kubernetes client:", err)
		os.Exit(1)
	}

	resources, err := FetchClusterDataTwo(clientset)
	if err != nil {
		fmt.Println("Failed to fetch data from Kubernetes cluster:", err)
		os.Exit(1)
	}

	// for simplicity, let's pick the first node we see
	resources = []NodeResourceInfo{resources[0]}
	for _, resource := range resources {
		fmt.Printf("NodeName: %s\n", resource.NodeName)
		fmt.Printf("CPU: %f\n", resource.CPU)
		fmt.Printf("Memory: %f\n", resource.Memory)
		fmt.Printf("MaxPods: %d\n", resource.MaxPods)
	}

}
