package mapping

type ImageMapping struct {
	Mapping map[string]string
}

func (im *ImageMapping) Set(url string, imageName string) {
	im.Mapping[url] = imageName
}

func (im *ImageMapping) Remove(url string) {
	delete(im.Mapping, url)
}
