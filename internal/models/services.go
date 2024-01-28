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
	Image       int
	Network     string
	Port        string

	Created time.Time
}

func (s *Service) Url() string {
	return fmt.Sprintf("http://%s:%s", s.Name, s.Port)
}

type ServiceModelInterface interface {
	Insert(name string, hosts []string, image int, network string, port string) (int, error)
	Get(id int) (*Service, error)
}

type ServiceModel struct {
	DB *sql.DB
}

func (m *ServiceModel) Insert(name string, hosts []string, image int, network string, port string) (int, error) {
	stmt := `INSERT INTO services (name, hosts, image, network, port) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	var id int
	err := m.DB.QueryRow(stmt, name, hosts, image, network, port).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (m *ServiceModel) Get(id int) (*Service, error) {
	stmt := `SELECT id, name, hosts, status, container_id, image, network, port FROM services WHERE id = $1`
	var s Service
	err := m.DB.QueryRow(stmt, id).Scan(&s.ID, &s.Name, &s.Hosts, &s.Status, &s.ContainerId, &s.Image, &s.Network, &s.Port)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
