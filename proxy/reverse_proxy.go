package proxy

import "github.com/sirupsen/logrus"

// This struct will maintain all the routing info
type ReverseProxy struct {
	Routes map[string][]string
}

func (r *ReverseProxy) Add(url string, containerName string) {
	allContainers := r.Routes[url]
	allContainers = append(allContainers, containerName)
	r.Routes[url] = allContainers
	logrus.Infof("Container %s added for %s route", containerName, url)
}

func (r *ReverseProxy) RemoveRoute(url string) {
	delete(r.Routes, url)
	logrus.Infof("Route %s removed!", url)
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
