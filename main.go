package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	// dockercontainer "github.com/AnkurTiwari21/DockerContainer"
	"github.com/AnkurTiwari21/containerhandler"
	"github.com/AnkurTiwari21/mapping"
	proxy "github.com/AnkurTiwari21/proxy"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func main() {
	//make any instance of the reverse proxy
	rp := proxy.ReverseProxy{
		Routes: map[string][]string{
			"localhost:8080": {"core"},
		},
		MatchMaking: map[string]int{
			"localhost:8080": 0,
		},
		RequestPerContainerPerSecond: map[string]int{
			"core": 0,
		},
	}

	im := mapping.ImageMapping{
		Mapping: map[string]string{
			"localhost:8080": "testserver",
		},
	}

	pm := mapping.PortMapping{
		Mapping: map[string]int{
			"localhost:8080": 5050,
		},
	}

	//basic http listener to listen at all the path and we will redirect the traffic based on subdomain
	r := gin.Default()
	// containerhandler.MonitorContainerWithID("core")
	r.Any("/*path", func(c *gin.Context) {
		// check if this domain is registered in the proxy
		requestedHost := c.Request.Host
		path := c.Request.RequestURI
		// hostArray := strings.Split(requestedHost, ":")

		logrus.Info(requestedHost)

		if rp.Routes[requestedHost] != nil {
			matchMakingAndCommunicate(c, requestedHost, path, &rp, &im, &pm)
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message": "route not found",
			})
		}
	})

	r.Run(":8080")
}

func matchMakingAndCommunicate(c *gin.Context, requestedHost string, path string, rp *proxy.ReverseProxy, im *mapping.ImageMapping, pm *mapping.PortMapping) {
	// for {

	// 	rp.FindMatch(requestedHost, im)
	// }
	targetContainer := rp.FindMatch(requestedHost, im)
	logrus.Infof("| Routing request to conatiner %s |", targetContainer)
	rp.View()
	portStr := strconv.Itoa(pm.Mapping[requestedHost])
	targetAddress := "http://" + targetContainer + ":" + portStr + path
	logrus.Info("target is ", targetAddress)
	client := http.Client{}

	req, err := http.NewRequest(c.Request.Method, targetAddress, c.Request.Body)
	if err != nil {
		logrus.Errorf("Error creating request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	for header, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(header, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("Error forwarding request: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to forward request"})
		return
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("Error reading response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read server response"})
		return
	}

	if path == "/create" {
		var respBody map[string]interface{}
		if err := json.Unmarshal(responseBody, &respBody); err != nil {
			logrus.Errorf("Error unmarshalling response: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process response"})
			return
		}

		// Process the "CREATE CONTAINER" operation
		if op, ok := respBody["operation"].(string); ok && op == "CREATE CONTAINER" {
			logrus.Info("Performing CREATE CONTAINER operation")

			// Generate container ID and start the container
			cid := uuid.New()
			containerId := containerhandler.RunContainerFromImageInBackground("testserver2", "ankur-net", cid.String())
			logrus.Infof("UUID: %s, Container ID: %s", cid, containerId)

			//register it in the reverse proxy
			rp.Add(containerId+"."+requestedHost, cid.String())
			im.Set(containerId+"."+requestedHost, "testserver2")

			//the initial pointer will be at 0th index
			rp.Mu.Lock()
			rp.MatchMaking[containerId+"."+requestedHost] = 0
			rp.Mu.Unlock()

			rp.View()

			if containerId != "" {
				respBody["id"] = containerId
			} else {
				logrus.Error("Error occurred while creating container")
			}
		}

		modifiedResponseBody, err := json.Marshal(respBody)
		if err != nil {
			logrus.Errorf("Error marshalling modified response: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process response"})
			return
		}

		responseBody = modifiedResponseBody
		logrus.Info("Modified Response:", string(responseBody))
	}

	for header, values := range resp.Header {
		for _, value := range values {
			c.Header(header, value)
		}
	}

	// Set the content length beacuse we added id in body and write the response body
	c.Header("Content-Length", fmt.Sprintf("%d", len(responseBody)))
	c.Status(resp.StatusCode)
	_, err = c.Writer.Write(responseBody)
	if err != nil {
		logrus.Errorf("Error writing response: %v", err)
	}
	for key, val := range rp.RequestPerContainerPerSecond {
		logrus.Infof("--container %s = cnt %s----", key, val)
	}
}
