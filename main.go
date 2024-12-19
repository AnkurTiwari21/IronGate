package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	// dockercontainer "github.com/AnkurTiwari21/DockerContainer"
	proxy "github.com/AnkurTiwari21/Proxy"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// "github.com/sirupsen/logrus"

func main() {
	//make any instance of the reverse proxy
	rp := proxy.ReverseProxy{
		Routes: map[string][]string{
			"localhost": {"my-container"},
		},
	}
	//basic http listener to listen at all the path and we will redirect the traffic based on subdomain
	r := gin.Default()

	r.Any("/*path", func(c *gin.Context) {
		// check if this domain is registered in the proxy
		requestedHost := c.Request.Host
		path := c.Request.RequestURI
		hostArray := strings.Split(requestedHost, ":")

		logrus.Info(requestedHost)

		if rp.Routes[hostArray[0]] != nil {
			matchMakingAndCommunicate(c, requestedHost, path)
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message": "route not found",
			})
		}
	})

	r.Run(":8080")
	// dockercontainer.ListImages()
	// dockercontainer.RunContainerFromImageInBackground()
	// dockercontainer.ListContainer()
	// dockercontainer.StopContainerByIdOrName("core")
	// dockercontainer.RunContainerFromImageInBackground()
}

func matchMakingAndCommunicate(c *gin.Context, requestedHost string, path string) {
	// Define the address of the Gin server
	address := "http://localhost:5050" // Use HTTP for communication

	// Send an HTTP GET request
	resp, err := http.Get(address)
	if err != nil {
		fmt.Printf("Error sending HTTP request: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to communicate with the server",
		})
		return
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read server response",
		})
		return
	}

	fmt.Printf("Received response from server: %s\n", string(body))
	c.JSON(http.StatusOK, gin.H{
		"message":         "route found",
		"host":            requestedHost,
		"path":            path,
		"server_response": string(body),
	})
}
