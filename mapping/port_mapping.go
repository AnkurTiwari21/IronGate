package mapping

type PortMapping struct {
	Mapping map[string]int
}

func (pm *PortMapping) Set(url string, port int) {
	pm.Mapping[url] = port
}

func (pm *PortMapping) Remove(url string) {
	delete(pm.Mapping, url)
}
