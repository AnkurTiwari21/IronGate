package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	// dockercontainer "github.com/AnkurTiwari21/DockerContainer"
	"github.com/AnkurTiwari21/containerhandler"
	"github.com/AnkurTiwari21/mapping"
	"github.com/AnkurTiwari21/migration"
	proxy "github.com/AnkurTiwari21/proxy"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func getClientIPByHeaders(req *http.Request) (ip string, err error) {

	// Client could be behid a Proxy, so Try Request Headers (X-Forwarder)
	ipSlice := []string{}

	ipSlice = append(ipSlice, req.Header.Get("X-Forwarded-For"))
	ipSlice = append(ipSlice, req.Header.Get("x-forwarded-for"))
	ipSlice = append(ipSlice, req.Header.Get("X-FORWARDED-FOR"))

	for _, v := range ipSlice {
		logrus.Infof("debug: client request header check gives ip: %v", v)
		if v != "" {
			return v, nil
		}
	}
	err = errors.New("error: Could not find clients IP address from the Request Headers")
	return "", err

}

func main() {
	//make any instance of the reverse proxy
	rp := proxy.ReverseProxy{
		Routes: map[string][]string{
			"localhost:8080": {},
		},
		MatchMaking: map[string]int{
			"localhost:8080": 0,
		},
		RequestPerContainerPerSecond: map[string]int{},
	}

	containerhandler.RunContainerFromImageInBackground("testserver", "ankur-net", "core")
	rp.Add("localhost:8080", "core")

	rp.Mu.Lock()
	rp.RequestPerContainerPerSecond["core"] = 0
	rp.Mu.Unlock()

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

	//perform cleanup of unused container here
	//using go routine to do process async
	//using time.NewTicker to do in a constant interval
	//default time for checking is 5min
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				logrus.Info("Checking for unused containers....")
				for url, containers := range rp.Routes {
					containerToBeRemoved, avgCPUUsage := containerhandler.ScaleDownContainers(containers)
					if containerToBeRemoved != "" {
						err := containerhandler.StopContainerByIdOrName(containerToBeRemoved)
						logrus.WithFields(logrus.Fields{
							"container": containerToBeRemoved,
							"avg_cpu":   avgCPUUsage,
						}).Info("Scaling down container due to low average CPU utilization")

						if err != nil {
							logrus.Error("Error stoping container | err ", err)
						}
					}
					rp.RemoveContainer(url, containerToBeRemoved)
				}
			}
		}
	}()

	//after succesfull setup init a redis connection to implement IP based blocking to prevent DDOS
	err := godotenv.Load(".env")
	if err != nil {
		logrus.Error("Error loading .env file")
		return
	}
	addr := os.Getenv("REDIS_ADDRESS")
	pass := os.Getenv("REDIS_PASSWORD")
	redisClient := migration.InitRedisClient(addr, pass)

	//basic http listener to listen at all the path and we will redirect the traffic based on subdomain
	r := gin.Default()
	// containerhandler.MonitorContainerWithID("core")
	r.Any("/*path", func(c *gin.Context) {
		// check if this domain is registered in the proxy
		requestedHost := c.Request.Host
		path := c.Request.RequestURI

		logrus.Info(requestedHost)
		userIP, err := getClientIPByHeaders(c.Request)
		if err != nil {
			logrus.Error("error in getting client ip | err ", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "try again",
			})
			return
		}
		logrus.Info("client ip ", userIP)
		attemptsMadeTillNow, err := redisClient.Get(context.Background(), userIP).Result()

		if err == redis.Nil {
			//ip not present in redis
			redisClient.Set(context.Background(), userIP, 1, 60*time.Second)
			if rp.Routes[requestedHost] != nil {
				matchMakingAndCommunicate(c, requestedHost, path, &rp, &im, &pm)
			} else {
				c.JSON(http.StatusOK, gin.H{
					"message": "route not found",
				})
			}
		} else if err != nil {
			logrus.Error("error getting data form redis | err ", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Try again after some time.",
			})
		} else {
			attemptsTillNow, err := strconv.Atoi(attemptsMadeTillNow)
			if err != nil {
				logrus.Error("error converting attempts to int | err ", err)
				return
			}
			//default check for 100 req/min/ip
			if attemptsTillNow >= 100 {
				redisClient.Set(context.Background(), userIP, attemptsTillNow, 5*60*time.Second)
				c.JSON(http.StatusOK, gin.H{
					"message": "You have been blocked! Try after some time.",
				})
			} else {
				redisClient.Set(context.Background(), userIP, attemptsTillNow+1, 60*time.Second)
				if rp.Routes[requestedHost] != nil {
					matchMakingAndCommunicate(c, requestedHost, path, &rp, &im, &pm)
				} else {
					c.JSON(http.StatusOK, gin.H{
						"message": "route not found",
					})
				}
			}
		}

	})

	r.Run(":8080")
}

func matchMakingAndCommunicate(c *gin.Context, requestedHost string, path string, rp *proxy.ReverseProxy, im *mapping.ImageMapping, pm *mapping.PortMapping) {
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
