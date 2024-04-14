package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/haidousm/delne/internal/models"
)

type Client struct {
	client *client.Client
	logger *slog.Logger
}

func NewClient(logger *slog.Logger) (*Client, error) {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	dLogger := logger.With("source", "docker")
	return &Client{client: client, logger: dLogger}, nil
}

func (c *Client) pullImage(image models.Image) error {
	reader, err := c.client.ImagePull(context.Background(), image.String(), types.ImagePullOptions{})
	if err != nil {
		return err
	}

	defer reader.Close()

	decoder := json.NewDecoder(reader)
	for {
		var progressMessage jsonmessage.JSONMessage
		if err := decoder.Decode(&progressMessage); err != nil {
			if err == io.EOF {
				// EOF is expected when the pull is complete
				return nil
			}

			// if err, try to skip the message and continue
			decoder.Buffered().Read(make([]byte, 1))
			continue
		}
		if progressMessage.Error != nil {
			return fmt.Errorf(progressMessage.Error.Message)
		}
		c.logger.Debug("pulling image", "status", progressMessage.Status)
	}
}

func (c *Client) listImages() []types.ImageSummary {
	list, err := c.client.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		panic(err)
	}
	return list
}

func (c *Client) imageExists(image models.Image) bool {
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

func (c *Client) CreateContainer(service models.Service, image models.Image) (container.CreateResponse, error) {

	c.logger.Debug("creating container", "service", service.Name, "image", image.String())
	if err := c.pullImage(image); err != nil {
		return container.CreateResponse{}, err
	}
	c.logger.Debug("image pulled", "service", service.Name, "image", image.String())
	if err := c.createNetwork(*service.Network); err != nil {
		return container.CreateResponse{}, err
	}
	c.logger.Debug("network created", "service", service.Name, "network", *service.Network)

	config := &container.Config{
		Image: image.String(),
	}
	hostConfig := &container.HostConfig{
		NetworkMode: container.NetworkMode(*service.Network),
	}

	if service.EnvironmentVariables != nil && len(*service.EnvironmentVariables) > 0 {
		env := []string{}
		for k, v := range *service.EnvironmentVariables {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		config.Env = env
	}

	resp, err := c.client.ContainerCreate(context.Background(), config, hostConfig, nil, nil, service.Name)
	c.logger.Debug("container created", "service", service.Name, "container", resp.ID)
	if err != nil {
		return container.CreateResponse{}, err
	}
	return resp, nil
}

func (c *Client) StartContainer(service models.Service) error {
	if service.ContainerId == nil {
		return errors.New("container id is empty")
	}

	err := c.client.ContainerStart(context.Background(), *service.ContainerId, types.ContainerStartOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) StopContainer(service models.Service) error {
	if service.ContainerId == nil {
		return errors.New("container id is empty")
	}

	err := c.client.ContainerStop(context.Background(), *service.ContainerId, container.StopOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) RemoveContainer(service models.Service) error {
	if service.ContainerId == nil {
		return nil
	}

	return c.RemoveContainerById(*service.ContainerId)
}

func (c *Client) RemoveContainerById(id string) error {
	err := c.client.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) inspectContainer(service models.Service) (types.ContainerJSON, error) {
	if service.ContainerId == nil {
		return types.ContainerJSON{}, nil
	}

	resp, err := c.client.ContainerInspect(context.Background(), *service.ContainerId)
	if err != nil {
		return types.ContainerJSON{}, err
	}
	return resp, nil
}

func (c *Client) GetContainerPorts(service models.Service) []string {
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

func (c *Client) GetContainerEnv(service models.Service) map[string]string {
	env := map[string]string{}
	resp, err := c.inspectContainer(service)
	if err != nil {
		return env
	}

	for _, e := range resp.Config.Env {
		pair := strings.Split(e, "=")
		env[pair[0]] = pair[1]
	}
	return env
}

func (c *Client) ListContainers() ([]types.Container, error) {
	containers, err := c.client.ContainerList(context.Background(), types.ContainerListOptions{})
	return containers, err
}
