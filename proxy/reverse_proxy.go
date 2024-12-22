package proxy

import (
	"sync"

	"github.com/AnkurTiwari21/containerhandler"
	"github.com/AnkurTiwari21/mapping"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// This struct will maintain all the routing info
type ReverseProxy struct {
	Routes                       map[string][]string
	MatchMaking                  map[string]int
	RequestPerContainerPerSecond map[string]int
	Mu                           sync.Mutex
}

func (r *ReverseProxy) Add(url string, containerName string) {
	r.Mu.Lock()
	allContainers := r.Routes[url]
	allContainers = append(allContainers, containerName)
	r.Routes[url] = allContainers
	r.Mu.Unlock()
	logrus.Infof("Container %s added for %s route", containerName, url)
}

func (r *ReverseProxy) RemoveRoute(url string) {
	r.Mu.Lock()
	delete(r.Routes, url)
	r.Mu.Unlock()
	logrus.Infof("Route %s removed!", url)
}

func (r *ReverseProxy) RemoveContainer(url string, containerName string) {
	r.Mu.Lock() // Ensure thread-safety if concurrent access is possible
	defer r.Mu.Unlock()

	containers := r.Routes[url]
	for i, val := range containers {
		if val == containerName {
			// Remove the container by slicing
			r.Routes[url] = append(containers[:i], containers[i+1:]...)
			return
		}
	}
}

func (r *ReverseProxy) Find(url string) bool {
	for key, _ := range r.Routes {
		if key == url {
			return true
		}
	}
	return false
}

func (r *ReverseProxy) View() {
	for key, values := range r.Routes {
		logrus.Info("route --> ", key)
		for _, value := range values {
			logrus.Info("containers are ", value)
		}
	}
}

func (r *ReverseProxy) FindMatch(url string, imageMapping *mapping.ImageMapping) string {
	// using a resource based alocation method
	// find avg cpu usage for that domain
	// if avg cpu usage > 80 --> spin up another conatiner and route traffic there
	// otherwise route traffic to coatiner with lowest cpu usage
	// return the container id

	avgCPUUsage := float64(0)
	var containerWithMinCPUUsage string
	minCPUUsage := float64(1000)

	for _, conatiner := range r.Routes[url] {
		cpuUsage, err := containerhandler.MonitorContainerWithID(conatiner)
		if err != nil {
			logrus.Errorf("error getting stats for container : %s | err ", conatiner)
			logrus.Error("error is ", err)
		} else {
			avgCPUUsage += (cpuUsage)
			if cpuUsage <= minCPUUsage {
				// logrus.Info(cpuUsage)
				if cpuUsage == minCPUUsage {
					if containerWithMinCPUUsage != "" && r.RequestPerContainerPerSecond[conatiner] <= r.RequestPerContainerPerSecond[containerWithMinCPUUsage] {
						containerWithMinCPUUsage = conatiner
					}
				} else {
					minCPUUsage = cpuUsage
					containerWithMinCPUUsage = conatiner
				}
			}

		}
	}
	avgCPUUsage = avgCPUUsage / float64(len(r.Routes[url]))
	if avgCPUUsage > 80 {
		conatinerUUId := uuid.New()
		containerhandler.RunContainerFromImageInBackground(imageMapping.Mapping[url], "ankur-net", conatinerUUId.String())
		r.Add(url, conatinerUUId.String())
		r.Mu.Lock()
		r.RequestPerContainerPerSecond[conatinerUUId.String()] += 1
		r.Mu.Unlock()
		return conatinerUUId.String()
	}
	r.Mu.Lock()
	r.RequestPerContainerPerSecond[containerWithMinCPUUsage] += 1
	r.Mu.Unlock()
	return containerWithMinCPUUsage
	// return ""
}
