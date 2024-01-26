package docker

import (
	"testing"

	"github.com/docker/docker/api/types/container"
)

func TestPullImage(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	image := Image{
		Repository: "_",
		Name:       "alpine",
		Tag:        "latest",
	}

	err = client.pullImage(image)
	if err != nil {
		t.Fatal(err)
	}

	images := client.listImages()
	if len(images) == 0 {
		t.Fatal("No images found")
	}

	if !client.imageExists(image) {
		t.Fatal("Image not found")
	}

	t.Log("Image pulled successfully")
}

func TestCreateContainer(t *testing.T) {

	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	image := Image{
		Repository: "_",
		Name:       "alpine",
		Tag:        "latest",
	}

	service := Service{
		Name:    "test-alpine",
		Image:   image,
		Network: "test-network",
	}

	resp, err := client.CreateContainer(service)
	if err != nil {
		t.Fatal(err)
	}

	if resp.ID == "" {
		t.Fatal("Container ID not found")
	}

	t.Log("Container created successfully")
	service.ContainerId = resp.ID
	insp, err := client.inspectContainer(service)
	if err != nil {
		t.Fatal(err)
	}

	if insp.HostConfig.NetworkMode != container.NetworkMode(service.Network) {
		t.Fatal("Container network not set")
	}

	t.Log("Container network set successfully")
	err = client.RemoveContainer(service)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStartContainer(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	image := Image{
		Repository: "_",
		Name:       "alpine",
		Tag:        "latest",
	}

	service := Service{
		Name:    "test-alpine",
		Image:   image,
		Network: "test-network",
	}

	resp, err := client.CreateContainer(service)
	if err != nil {
		t.Fatal(err)
	}

	if resp.ID == "" {
		t.Fatal("Container ID not found")
	}

	t.Log("Container created successfully")
	service.ContainerId = resp.ID
	err = client.StartContainer(service)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Container started successfully")
	err = client.StopContainer(service)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Container stopped successfully")
	err = client.RemoveContainer(service)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Container removed successfully")
}
