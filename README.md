# Hosted Control Plane Sizing Calculator

This repository contains a command-line tool for calculating the sizing requirements of self-managed hosted control planes (HCP) for Red Hat OpenShift.

The tool factors in CPU, memory, and ETCD storage based on the workload characteristics and the expected Queries Per Second (QPS) rate to the KubeAPI server to give you the best estimates based on studies performed by performance and scale teams.

## Features

- Calculate the maximum number of HCPs that can be hosted on a worker node.
- Estimate CPU and memory requirements based on request-based or load-based sizing methods.
- Estimate ETCD storage requirements based on the number of pods.

## Prerequisites

To use this tool, you need to have Go installed on your machine. Visit [Go's official documentation](https://golang.org/doc/install) for installation instructions.

## Usage

1. Clone the repository to your local machine.
2. Navigate to the cloned directory and build the tool with Go:

```sh
go build -o hcp-sizer
```

3. Run the calculator `./hcp-sizer`
4. Follow the interactive prompts to enter your cluster's specifications and choose the calculation method.


## Estimating API server QPS

To accurately size your HCP, you'll need to estimate the QPS rate for your cluster. Run the following query on your existing cluster:

```sh
kubectl get --raw /metrics | grep -E 'apiserver_request_total|apiserver_request_duration_seconds_count'
```

or 

```sh
sum(rate(apiserver_request_total{namespace=~"clusters-$name*"}[2m])) by (namespace)
```
This Prometheus query will provide you with the rate of queries to your API server from your cluster, which informs sizer's load-based method.

### QPS Categories

Typical QPS categories are as follows for reference:

| Category  | Description     |
|-----------|-----------------|
| Low       | 0-1000 QPS      |
| Medium    | 1000-5000 QPS   |
| High      | 5000-10000 QPS  |
| Very High | 10000-20000 QPS |

### Server Categories

Typical server categories are as follows for reference:

| **Category**            | **Description**                                                                                                               |
|-------------------------|-------------------------------------------------------------------------------------------------------------------------------|
| **Entry-Level Servers** | **CPU**: Typically 4 to 8 cores.<br/>**Memory**: Ranges from 8GB to 32GB of RAM.                                              |
| **Mid-Range Servers**   | **CPU**: Generally have between 12 to 24 cores.<br/>**Memory**: Equipped with 64GB to 256GB of RAM.                           |
| **High-End Servers**    | **CPU**: Could have 32 cores or more, potentially with multiple CPUs.<br/>**Memory**: From 256GB to several terabytes of RAM. |


# Demo
[HCP Sizer Demo](https://www.youtube.com/watch?v=VIDEO_ID)

# Contributing
Contributions to the HCP Sizing Calculator are welcome! Please read our contributing guidelines to get started.

# License
This tool is open source and available under the MIT License.

# Acknowledgments
This tool is a convinient wrapper around the [data and formulas](https://access.redhat.com/documentation/en-us/red_hat_advanced_cluster_management_for_kubernetes/2.9/html/clusters/cluster_mce_overview#hosted-sizing-guidance) provided by OpenShift Performance and Scale 


