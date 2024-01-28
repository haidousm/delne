package models

import "regexp"

type Image struct {
	Repository string
	Name       string
	Tag        string
}

func (i *Image) String() string {
	if i.Repository == "_" {
		return i.Name + ":" + i.Tag
	}
	return i.Repository + "/" + i.Name + ":" + i.Tag
}

func (i *Image) ParseString(image string) {
	if image == "" {
		return
	}

	regex := regexp.MustCompile(`^(.+)\/(.+):(.+)$|^(.+):(.+)$|^(.+)\/(.+)|^(.+)$`)
	match := regex.FindStringSubmatch(image)

	if len(match) > 0 {
		if match[1] != "" {
			i.Repository = match[1]
			i.Name = match[2]
			i.Tag = match[3]
		} else if match[4] != "" {
			i.Repository = "_"
			i.Name = match[4]
			i.Tag = match[5]
		} else if match[6] != "" {
			i.Repository = match[6]
			i.Name = match[7]
		} else if match[8] != "" {
			i.Repository = "_"
			i.Name = match[8]
			i.Tag = "latest"
		}
	}
}
