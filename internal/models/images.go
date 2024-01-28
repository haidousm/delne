package models

import (
	"database/sql"
	"regexp"
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

type ImageModelInterface interface {
	Insert(repository string, name string, tag string) (int, error)
	Get(id int) (*Image, error)
}

type ImageModel struct {
	DB *sql.DB
}

func (m *ImageModel) Insert(repository string, name string, tag string) (int, error) {
	stmt := `INSERT INTO images (repository, name, tag) VALUES ($1, $2, $3) RETURNING id`
	var id int
	err := m.DB.QueryRow(stmt, repository, name, tag).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (m *ImageModel) Get(id int) (*Image, error) {
	stmt := `SELECT id, repository, name, tag FROM images WHERE id = $1`
	var i Image
	err := m.DB.QueryRow(stmt, id).Scan(&i)
	if err != nil {
		return nil, err
	}
	return &i, nil
}
