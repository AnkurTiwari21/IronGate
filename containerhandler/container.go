package containerhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

// working
func ListContainer() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, containertypes.ListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Printf("Id: %s, Container Name:%s \n", container.ID, container.Names[0][1:])
	}
}

// working
func ListImages() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		panic(err)
	}

	for _, image := range images {
		fmt.Println(image.ID)
	}
}

// working
func RunContainerFromImageInBackground(image_name string, networkName string, container_name string) string {
	ctx := context.Background()
	// Create a Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Printf("Error creating Docker client: %v\n", err)
		return ""
	}

	// Define the name of the existing image
	imageName := image_name // Replace with your image name

	// Define your custom network name
	customNetwork := networkName // Replace with your network name

	// Check if the network exists
	networkList, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		fmt.Printf("Error listing Docker networks: %v\n", err)
		return ""
	}

	networkExists := false
	for _, network := range networkList {
		if network.Name == customNetwork {
			networkExists = true
			break
		}
	}

	if !networkExists {
		fmt.Printf("Network %s does not exist. Please create it first.\n", customNetwork)
		return ""
	}

	// Create the container
	containerConfig := &container.Config{
		Image: imageName, // Specify the image name
	}
	hostConfig := &container.HostConfig{
		AutoRemove: true, // Automatically remove the container when it stops
	}
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			customNetwork: {}, // Attach the container to the custom network
		},
	}

	containerName := container_name // Name of the container
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		fmt.Printf("Error creating container: %v\n", err)
		return ""
	}

	fmt.Printf("Container %s created with ID: %s\n", containerName, resp.ID)

	// Start the container
	if err := cli.ContainerStart(ctx, resp.ID, containertypes.StartOptions{}); err != nil {
		fmt.Printf("Error starting container: %v\n", err)
		return ""
	}

	time.Sleep(2 * time.Second) //for graceful start in case for some delay
	fmt.Printf("Container %s started successfully in network %s.\n", containerName, customNetwork)
	return resp.ID
}

// working
// pass the container id or container name
func StopContainerByIdOrName(containerId string) error {
	ctx := context.Background()

	// Create a Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logrus.Errorf("Error creating Docker client: %v\n", err)
		return err
	}

	// Define the name or ID of the container to stop
	containerID := containerId // Replace with your container's name or ID

	// Stop the container
	timeout := 10 // Graceful shutdown timeout
	if err := cli.ContainerStop(ctx, containerID, containertypes.StopOptions{Timeout: &timeout}); err != nil {
		logrus.Errorf("Error stopping container %s: %v\n", containerID, err)
		return err
	}

	logrus.Infof("Container %s stopped successfully.\n", containerID)
	return nil
}

func MonitorAllContainers(toStream bool) {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(ctx, containertypes.ListOptions{})
	if err != nil {
		panic(err)
	}

	// Monitor stats for each container
	for _, container := range containers {
		go func(containerID string) {
			stats, err := cli.ContainerStats(ctx, containerID, toStream)
			if err != nil {
				fmt.Printf("Error fetching stats for container %s: %v\n", containerID, err)
				return
			}
			defer stats.Body.Close()

			decoder := json.NewDecoder(stats.Body)
			for {
				var stat types.Stats
				if err := decoder.Decode(&stat); err == io.EOF {
					break
				} else if err != nil {
					fmt.Printf("Error decoding stats for container %s: %v\n", containerID, err)
					break
				}

				// Calculate CPU and Memory Usage
				cpuDelta := float64(stat.CPUStats.CPUUsage.TotalUsage) - float64(stat.PreCPUStats.CPUUsage.TotalUsage)
				systemDelta := float64(stat.CPUStats.SystemUsage) - float64(stat.PreCPUStats.SystemUsage)
				cpuPercent := (cpuDelta / systemDelta) * float64(len(stat.CPUStats.CPUUsage.PercpuUsage)) * 100.0

				memPercent := float64(stat.MemoryStats.Usage) / float64(stat.MemoryStats.Limit) * 100.0

				fmt.Printf("Container: %s | CPU: %.2f%% | Memory: %.2f%%\n", containerID, cpuPercent, memPercent)
			}
		}(container.ID)
	}

	// Keep the program running
	select {}
}

func MonitorContainerWithID(containerId string) (float64, error) {
	// Initialize the Docker client
	cli, err := client.NewClientWithOpts(client.WithVersion("1.41"))
	if err != nil {
		logrus.Errorf("Error creating Docker client: %v", err)
		return 0, nil
	}

	containerID := containerId // Replace with your container ID

	// Fetch the initial stats
	stats1, err := cli.ContainerStats(context.Background(), containerID, false)
	if err != nil {
		logrus.Errorf("Error fetching stats: %v", err)
		return 0, nil
	}
	defer stats1.Body.Close()

	// Fetch stats again after a short delay (1 second or more)
	time.Sleep(1 * time.Second)

	stats2, err := cli.ContainerStats(context.Background(), containerID, false)
	if err != nil {
		logrus.Errorf("Error fetching stats: %v", err)
		return 0, nil
	}
	defer stats2.Body.Close()

	var stat1, stat2 types.StatsJSON
	if err := json.NewDecoder(stats1.Body).Decode(&stat1); err != nil {
		logrus.Errorf("Error decoding stats1: %v", err)
	}
	if err := json.NewDecoder(stats2.Body).Decode(&stat2); err != nil {
		logrus.Errorf("Error decoding stats2: %v", err)
	}

	// Calculate the CPU usage
	cpuDelta := float64(stat2.CPUStats.CPUUsage.TotalUsage - stat1.CPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stat2.CPUStats.SystemUsage - stat1.CPUStats.SystemUsage)
	onlineCPUs := float64(stat2.CPUStats.OnlineCPUs)

	// Calculate CPU percentage
	cpuPercentage := (cpuDelta / systemDelta) * onlineCPUs * 100

	fmt.Printf("CPU Usage: %.2f%% for container %s\n", cpuPercentage, containerID)
	return cpuPercentage, nil
}

func ScaleDownContainers(containers []string) (string, float64) {
	avgCPUUsage := float64(0)
	var containerWithMinCPUUsage string
	var minCPUUsage = float64(100)
	for _, conatinerId := range containers {
		//get cpu stats
		cpuUsage, err := MonitorContainerWithID(conatinerId)
		if err != nil {
			logrus.Error("error in getting stats for the container | err ", err)
		}
		avgCPUUsage += (cpuUsage)
		if cpuUsage <= minCPUUsage {
			minCPUUsage = cpuUsage
			containerWithMinCPUUsage = conatinerId
		}
	}
	if avgCPUUsage < 10 && len(containers) > 1 {
		//scale down the container with min cpu usage
		return containerWithMinCPUUsage, avgCPUUsage
	}
	return "", 0
}
