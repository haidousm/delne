package docker

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
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

func (i *Image) ParseString(image string) {
	if image == "" {
		return
	}

	/**
		const regex = /^(.+)\/(.+):(.+)$|^(.+):(.+)$|^(.+)\/(.+)|^(.+)$/;
	    const match = regex.exec(imageIdentifier);

	    if (match) {
	        const repository = match[1] ?? match[6] ?? "_";
	        const name = match[2] ?? match[4] ?? match[7] ?? match[8];
	        const tag = match[3] ?? match[5] ?? "latest";

	        return {
	            repository,
	            name,
	            tag,
	        };
	    }
	    throw new Error(`Invalid image identifier: ${imageIdentifier}`);
	*/

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

type Service struct {
	Name  string
	Hosts []string

	ContainerId string
	Image       Image
	Network     string
	Port        string
}

func (s *Service) Url() string {
	return fmt.Sprintf("http://%s:%s", s.Name, s.Port)
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

func (c *Client) networkExists(name string) bool {
	networks, err := c.client.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		panic(err)
	}
	for _, network := range networks {
		if network.Name == name {
			return true
		}
	}
	return false
}

func (c *Client) createNetwork(name string) error {
	if c.networkExists(name) {
		return nil
	}
	_, err := c.client.NetworkCreate(context.Background(), name, types.NetworkCreate{})
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) CreateContainer(service Service) (container.CreateResponse, error) {
	if err := c.pullImage(service.Image); err != nil {
		return container.CreateResponse{}, err
	}

	if err := c.createNetwork(service.Network); err != nil {
		return container.CreateResponse{}, err
	}

	resp, err := c.client.ContainerCreate(context.Background(), &container.Config{
		Image: service.Image.String(),
	}, &container.HostConfig{
		NetworkMode: container.NetworkMode(service.Network),
	}, nil, nil, service.Name)

	if err != nil {
		return container.CreateResponse{}, err
	}
	return resp, nil
}

func (c *Client) StartContainer(service Service) error {
	if service.ContainerId == "" {
		return nil
	}

	err := c.client.ContainerStart(context.Background(), service.ContainerId, types.ContainerStartOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) StopContainer(service Service) error {
	if service.ContainerId == "" {
		return nil
	}

	err := c.client.ContainerStop(context.Background(), service.ContainerId, container.StopOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) RemoveContainer(service Service) error {
	if service.ContainerId == "" {
		return nil
	}

	err := c.client.ContainerRemove(context.Background(), service.ContainerId, types.ContainerRemoveOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) inspectContainer(service Service) (types.ContainerJSON, error) {
	if service.ContainerId == "" {
		return types.ContainerJSON{}, nil
	}

	resp, err := c.client.ContainerInspect(context.Background(), service.ContainerId)
	if err != nil {
		return types.ContainerJSON{}, err
	}
	return resp, nil
}

func (c *Client) GetContainerPorts(service Service) []string {
	ports := []string{}
	resp, err := c.inspectContainer(service)
	if err != nil {
		return ports
	}

	for p := range resp.Config.ExposedPorts {
		ports = append(ports, string(p.Port()))
	}
	return ports
}
