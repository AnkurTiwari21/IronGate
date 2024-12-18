package dockercontainer

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
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

func createTar(filePath string) (io.Reader, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Dockerfile: %v", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat Dockerfile: %v", err)
	}

	header := &tar.Header{
		Name: "Dockerfile",
		Size: stat.Size(),
		Mode: int64(stat.Mode()),
	}
	if err := tw.WriteHeader(header); err != nil {
		return nil, fmt.Errorf("failed to write tar header: %v", err)
	}

	if _, err := io.Copy(tw, file); err != nil {
		return nil, fmt.Errorf("failed to copy Dockerfile to tar: %v", err)
	}

	return buf, nil
}

func BuildImage() {
	ctx := context.Background()

	// Create a Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Printf("Error creating Docker client: %v\n", err)
		return
	}

	// Define the path to the Dockerfile
	dockerfilePath := "./Dockerfile"

	// Create a tar archive containing the Dockerfile
	dockerfileTar, err := createTar(dockerfilePath)
	if err != nil {
		fmt.Printf("Error creating tar archive: %v\n", err)
		return
	}

	// Build the image
	imageName := "my-custom-image:latest"
	buildResponse, err := cli.ImageBuild(ctx, dockerfileTar, types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: "Dockerfile", // Name of the Dockerfile
		Remove:     true,         // Clean up intermediate containers
	})
	if err != nil {
		fmt.Printf("Error building Docker image: %v\n", err)
		return
	}
	defer buildResponse.Body.Close()

	// Print the build response logs
	io.Copy(os.Stdout, buildResponse.Body)
	fmt.Println("Docker image built successfully!")
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
func RunContainerFromImageInBackground() {
	ctx := context.Background()

	// Create a Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Printf("Error creating Docker client: %v\n", err)
		return
	}

	// Define the name of the existing image
	imageName := "testserver" // Replace with your image name

	// Create the container
	containerConfig := &container.Config{
		Image: imageName, // Specify the image name
	}
	hostConfig := &container.HostConfig{
		AutoRemove: true, // Automatically remove the container when it stops
	}

	containerName := "my-container" // Name of the container
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		fmt.Printf("Error creating container: %v\n", err)
		return
	}

	fmt.Printf("Container %s created with ID: %s\n", containerName, resp.ID)

	// Start the container
	if err := cli.ContainerStart(ctx, resp.ID, containertypes.StartOptions{}); err != nil {
		fmt.Printf("Error starting container: %v\n", err)
		return
	}

	fmt.Printf("Container %s started successfully.\n", containerName)
}

//working
func StopContainerById(containerId string) {
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
