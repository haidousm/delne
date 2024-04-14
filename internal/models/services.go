package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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
	ContainerId *string
	ImageID     *int
	Network     *string
	Port        *string

	EnvironmentVariables *map[string]string

	Created time.Time
}

func (s *Service) Url() string {
	if s.Port == nil {
		return fmt.Sprintf("http://%s", s.Name)
	}
	return fmt.Sprintf("http://%s:%s", s.Name, *s.Port)
}

type ServiceModelInterface interface {
	Insert(name string, hosts []string, image int, network string) (int, error)
	Get(id int) (*Service, error)
	GetAll() ([]*Service, error)

	GetByName(name string) (*Service, error)

	UpdateStatus(id int, status ServiceStatus) error
	UpdateContainerId(id int, containerId string) error
	UpdatePort(id int, port string) error

	UpdateEnvironmentVariables(id int, envVars map[string]string) error

	Delete(id int) error
}

type ServiceModel struct {
	DB *sql.DB
}

func (m *ServiceModel) Insert(name string, hosts []string, image_id int, network string) (int, error) {

	hostsCSV := ""
	for _, host := range hosts {
		hostsCSV += host + ","
	}
	stmt := `INSERT INTO services (name, hosts, image_id, network, status, environment_variables, created) VALUES ($1, $2, $3, $4, $5, "{}", datetime('now')) RETURNING id`
	var id int
	err := m.DB.QueryRow(stmt, name, hostsCSV, image_id, network, PULLING).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (m *ServiceModel) Get(id int) (*Service, error) {
	stmt := `SELECT id, name, hosts, status, container_id, image_id, network, port, environment_variables FROM services WHERE id = $1`
	var s Service
	hostsCSV := ""
	envVarsJSON := ""
	err := m.DB.QueryRow(stmt, id).Scan(&s.ID, &s.Name, &hostsCSV, &s.Status, &s.ContainerId, &s.ImageID, &s.Network, &s.Port, &envVarsJSON)
	if err != nil {
		return nil, err
	}
	s.Hosts = []string{}
	for _, host := range strings.Split(hostsCSV, ",") {
		if host != "" {
			s.Hosts = append(s.Hosts, host)
		}
	}

	s.EnvironmentVariables = &map[string]string{}
	if envVarsJSON != "" {
		err = json.Unmarshal([]byte(envVarsJSON), s.EnvironmentVariables)
		if err != nil {
			return nil, err
		}
	}

	return &s, nil
}

func (m *ServiceModel) GetAll() ([]*Service, error) {
	stmt := `SELECT id, name, hosts, status, container_id, image_id, network, port, environment_variables FROM services`
	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []*Service
	for rows.Next() {

		hostsCSV := ""
		envVarsJSON := ""

		var s Service
		err := rows.Scan(&s.ID, &s.Name, &hostsCSV, &s.Status, &s.ContainerId, &s.ImageID, &s.Network, &s.Port, &envVarsJSON)
		if err != nil {
			return nil, err
		}
		s.Hosts = []string{}
		for _, host := range strings.Split(hostsCSV, ",") {
			if host != "" {
				s.Hosts = append(s.Hosts, host)
			}
		}

		s.EnvironmentVariables = &map[string]string{}
		if envVarsJSON != "" {
			err = json.Unmarshal([]byte(envVarsJSON), s.EnvironmentVariables)
			if err != nil {
				return nil, err
			}
		}

		services = append(services, &s)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return services, nil
}

func (m *ServiceModel) GetByName(name string) (*Service, error) {
	stmt := `SELECT id, name, hosts, status, container_id, image_id, network, port, environment_variables FROM services WHERE name = $1`

	hostsCSV := ""
	envVarsJSON := ""
	var s Service
	err := m.DB.QueryRow(stmt, name).Scan(&s.ID, &s.Name, &hostsCSV, &s.Status, &s.ContainerId, &s.ImageID, &s.Network, &s.Port, &envVarsJSON)
	if err != nil {
		return nil, err
	}

	s.Hosts = []string{}
	for _, host := range strings.Split(hostsCSV, ",") {
		if host != "" {
			s.Hosts = append(s.Hosts, host)
		}
	}

	s.EnvironmentVariables = &map[string]string{}
	if envVarsJSON != "" {
		err = json.Unmarshal([]byte(envVarsJSON), s.EnvironmentVariables)
		if err != nil {
			return nil, err
		}
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

func (m *ServiceModel) UpdatePort(id int, port string) error {
	stmt := `UPDATE services SET port = $1 WHERE id = $2`
	_, err := m.DB.Exec(stmt, port, id)
	if err != nil {
		return err
	}
	return nil
}

func (m *ServiceModel) Delete(id int) error {
	stmt := `DELETE FROM services WHERE id = $1`
	_, err := m.DB.Exec(stmt, id)
	if err != nil {
		return err
	}
	return nil
}

func (m *ServiceModel) UpdateEnvironmentVariables(id int, envVars map[string]string) error {

	envVarsJSON, err := json.Marshal(envVars)
	if err != nil {
		return err
	}

	stmt := `UPDATE services SET environment_variables = $1 WHERE id = $2`
	_, err = m.DB.Exec(stmt, envVarsJSON, id)
	if err != nil {
		return err
	}
	return nil
}
