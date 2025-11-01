package models

import "time"

type DatabaseType string

const (
	DatabaseTypePostgres DatabaseType = "postgres"
	DatabaseTypeValkey   DatabaseType = "valkey"
)

type DatabaseStatus string

const (
	DatabaseStatusRunning  DatabaseStatus = "running"
	DatabaseStatusStopped  DatabaseStatus = "stopped"
	DatabaseStatusCreating DatabaseStatus = "creating"
	DatabaseStatusError    DatabaseStatus = "error"
)

type Database struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	Type             DatabaseType `json:"type"`
	ContainerID      string       `json:"container_id"`
	ContainerName    string       `json:"container_name"`
	VolumeName       string       `json:"volume_name"`
	Network          string       `json:"network"` // Deprecated: use VPC
	Port             int          `json:"port"`    // Deprecated: use PublishedPort
	InternalPort     int          `json:"internal_port"`
	Host             string       `json:"host"` // Deprecated: use localhost for published
	Username         string       `json:"username"`
	Password         string       `json:"password"`
	DatabaseName     string       `json:"database"`
	ConnectionString string       `json:"connection_string"`

	VPC                       string `json:"vpc"`
	Published                 bool   `json:"published"`
	PublishedPort             int    `json:"published_port"`
	InternalHostname          string `json:"internal_hostname"`
	PublishedConnectionString string `json:"published_connection_string"`

	LinkedApps []string `json:"linked_apps"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Status    DatabaseStatus `json:"status"`
}

type Registry struct {
	Databases []Database `json:"databases"`
}
