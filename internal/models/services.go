package models

import "fmt"

type ServiceStatus string

const (
	PULLING ServiceStatus = "PULLING"
	CREATED ServiceStatus = "CREATED"
	RUNNING ServiceStatus = "RUNNING"
	STOPPED ServiceStatus = "STOPPED"
	ERROR   ServiceStatus = "ERROR"
)

type Service struct {
	Name  string
	Hosts []string

	Status      ServiceStatus
	ContainerId string
	Image       Image
	Network     string
	Port        string
}

func (s *Service) Url() string {
	return fmt.Sprintf("http://%s:%s", s.Name, s.Port)
}
