package docker

import "testing"

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
