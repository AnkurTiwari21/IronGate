package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	proxy "github.com/AnkurTiwari21/Proxy"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	// "github.com/sirupsen/logrus"
	containertypes "github.com/docker/docker/api/types/container"
)

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
		fmt.Println(container.ID)
	}
}
func main() {
	//make any instance of the reverse proxy
	rp := proxy.ReverseProxy{
		Routes: map[string][]string{
			"localhost": {"core"},
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
			//TODO: implement a match making algorithm to find which container it should redirect the traffic
			c.JSON(http.StatusOK, gin.H{
				"message": "route found",
				"host":    requestedHost,
				"path":    path,
			})
		}else{
			c.JSON(http.StatusOK, gin.H{
				"message": "route not found",
			})
		}
	})

	r.Run(":8080")
}
