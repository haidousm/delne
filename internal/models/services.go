package models

import (
	"database/sql"
	"fmt"
	"time"
)

type ServiceStatus string

const (
	PULLING ServiceStatus = "PULLING"
	CREATED ServiceStatus = "CREATED"
	RUNNING ServiceStatus = "RUNNING"
	STOPPED ServiceStatus = "STOPPED"
	ERROR   ServiceStatus = "ERROR"
)

type Service struct {
	ID    int
	Name  string
	Hosts []string

	Status      ServiceStatus
	ContainerId string
	ImageID     int
	Network     string
	Port        string

	Created time.Time
}

func (s *Service) Url() string {
	return fmt.Sprintf("http://%s:%s", s.Name, s.Port)
}

type ServiceModelInterface interface {
	Insert(name string, hosts []string, image int, network string) (int, error)
	Get(id int) (*Service, error)
	GetAll() ([]*Service, error)

	GetByName(name string) (*Service, error)

	UpdateStatus(id int, status ServiceStatus) error
	UpdateContainerId(id int, containerId string) error
}

type ServiceModel struct {
	DB *sql.DB
}

func (m *ServiceModel) Insert(name string, hosts []string, image_id int, network string) (int, error) {
	stmt := `INSERT INTO services (name, hosts, image_id, network) VALUES ($1, $2, $3, $4) RETURNING id`
	var id int
	err := m.DB.QueryRow(stmt, name, hosts, image_id, network).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (m *ServiceModel) Get(id int) (*Service, error) {
	stmt := `SELECT id, name, hosts, status, container_id, image_id, network, port FROM services WHERE id = $1`
	var s Service
	err := m.DB.QueryRow(stmt, id).Scan(&s.ID, &s.Name, &s.Hosts, &s.Status, &s.ContainerId, &s.ImageID, &s.Network, &s.Port)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (m *ServiceModel) GetAll() ([]*Service, error) {
	stmt := `SELECT id, name, hosts, status, container_id, image_id, network, port FROM services`
	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []*Service
	for rows.Next() {
		var s Service
		err := rows.Scan(&s.ID, &s.Name, &s.Hosts, &s.Status, &s.ContainerId, &s.ImageID, &s.Network, &s.Port)
		if err != nil {
			return nil, err
		}
		services = append(services, &s)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return services, nil
}

func (m *ServiceModel) GetByName(name string) (*Service, error) {
	stmt := `SELECT id, name, hosts, status, container_id, image_id, network, port FROM services WHERE name = $1`
	var s Service
	err := m.DB.QueryRow(stmt, name).Scan(&s.ID, &s.Name, &s.Hosts, &s.Status, &s.ContainerId, &s.ImageID, &s.Network, &s.Port)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (m *ServiceModel) UpdateStatus(id int, status ServiceStatus) error {
	stmt := `UPDATE services SET status = $1 WHERE id = $2`
	_, err := m.DB.Exec(stmt, status, id)
	if err != nil {
		return err
	}
	return nil
}

func (m *ServiceModel) UpdateContainerId(id int, containerId string) error {
	stmt := `UPDATE services SET container_id = $1 WHERE id = $2`
	_, err := m.DB.Exec(stmt, containerId, id)
	if err != nil {
		return err
	}
	return nil
}
