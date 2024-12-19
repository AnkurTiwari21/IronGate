package containerhandler

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
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

	fmt.Printf("Container %s started successfully in network %s.\n", containerName, customNetwork)
	return resp.ID
}

// working
// pass the container id or container name
func StopContainerByIdOrName(containerId string) {
	ctx := context.Background()

	// Create a Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Printf("Error creating Docker client: %v\n", err)
		return
	}

	// Define the name or ID of the container to stop
	containerID := containerId // Replace with your container's name or ID

	// Stop the container
	timeout := 10 // Graceful shutdown timeout
	if err := cli.ContainerStop(ctx, containerID, containertypes.StopOptions{Timeout: &timeout}); err != nil {
		fmt.Printf("Error stopping container %s: %v\n", containerID, err)
		return
	}

	fmt.Printf("Container %s stopped successfully.\n", containerID)
}
