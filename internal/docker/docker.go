package docker

import (
	"context"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

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

type Client struct {
	client *client.Client
}

func NewClient() (*Client, error) {
	// create docker client
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	return &Client{client: client}, nil
}

func (c *Client) pullImage(image Image) error {
	reader, err := c.client.ImagePull(context.Background(), image.String(), types.ImagePullOptions{})
	if err != nil {
		return err
	}

	defer reader.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		time.Sleep(1 * time.Second)
		if err := reader.Close(); err != nil {
			panic(err)
		}
		if c.imageExists(image) {
			wg.Done()
		}
	}(&wg)
	wg.Wait()
	return nil
}

func (c *Client) listImages() []types.ImageSummary {
	list, err := c.client.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		panic(err)
	}
	return list
}

func (c *Client) imageExists(image Image) bool {
	images := c.listImages()
	for _, ims := range images {
		for _, tag := range ims.RepoTags {
			if tag == image.String() {
				return true
			}
		}
	}
	return false
}
