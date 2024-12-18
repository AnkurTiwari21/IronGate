package proxy

// This struct will maintain all the routing info
type ReverseProxy struct {
	Routes map[string][]string
}

func (r *ReverseProxy) Add(url string, containerName string) {

}

func (r *ReverseProxy) Remove(url string) {

}

func (r *ReverseProxy) Find(url string)  {
	// for _, val := range r.Routes {
	// 	if val == url {
	// 		return true
	// 	}
	// }
	// return false
}
