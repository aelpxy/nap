package models

import "time"

type AppStatus string

const (
	AppStatusRunning   AppStatus = "running"
	AppStatusStopped   AppStatus = "stopped"
	AppStatusDeploying AppStatus = "deploying"
	AppStatusFailed    AppStatus = "failed"
	AppStatusScaling   AppStatus = "scaling"
)

type BuildType string

const (
	BuildTypeDockerfile BuildType = "dockerfile"
	BuildTypeNixpacks   BuildType = "nixpacks"
	BuildTypePacketo    BuildType = "packeto"
)

type DeploymentStrategy string

const (
	DeploymentStrategyRecreate  DeploymentStrategy = "recreate"
	DeploymentStrategyRolling   DeploymentStrategy = "rolling"
	DeploymentStrategyBlueGreen DeploymentStrategy = "blue-green"
)

type DeploymentColor string

const (
	DeploymentColorBlue    DeploymentColor = "blue"
	DeploymentColorGreen   DeploymentColor = "green"
	DeploymentColorDefault DeploymentColor = "default"
)

type Application struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	VPC  string `json:"vpc"`

	Status         AppStatus `json:"status"`
	Instances      int       `json:"instances"`
	BuildType      BuildType `json:"build_type"`
	Buildpack      string    `json:"buildpack"`
	DockerfilePath string    `json:"dockerfile_path"`

	ContainerIDs []string `json:"container_ids"`
	ImageID      string   `json:"image_id"`

	Memory int     `json:"memory"` // MB per instance
	CPU    float64 `json:"cpu"`    // cores per instance
	Port   int     `json:"port"`

	InternalHostname string   `json:"internal_hostname"`
	Published        bool     `json:"published"`
	PublishedURL     string   `json:"published_url"` // https://{name}.nap.app
	PublishedDomain  string   `json:"published_domain"`
	CustomDomains    []string `json:"custom_domains"`
	PublishedPort    int      `json:"published_port"`
	SSLEnabled       bool     `json:"ssl_enabled"`
	SSLCertIssuer    string   `json:"ssl_cert_issuer"`
	SSLCertExpiry    string   `json:"ssl_cert_expiry"`

	LinkedDatabases []string `json:"linked_databases"`

	Volumes []Volume `json:"volumes,omitempty"`

	HealthCheckPath     string `json:"health_check_path"`
	HealthCheckInterval int    `json:"health_check_interval"`
	HealthCheckTimeout  int    `json:"health_check_timeout"`

	AutoScaleEnabled bool `json:"autoscale_enabled"`
	MinInstances     int  `json:"min_instances"`
	MaxInstances     int  `json:"max_instances"`
	TargetCPU        int  `json:"target_cpu"`
	TargetMemory     int  `json:"target_memory"`

	EnvVars map[string]string `json:"env_vars"`

	DeploymentStrategy DeploymentStrategy `json:"deployment_strategy"`
	DeploymentConfig   DeploymentConfig   `json:"deployment_config"`
	DeploymentState    DeploymentState    `json:"deployment_state"`
	DeploymentHistory  []DeploymentRecord `json:"deployment_history"`

	SourcePath     string    `json:"source_path"`
	BuildID        string    `json:"build_id"`
	CurrentVersion int       `json:"current_version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	LastDeployedAt time.Time `json:"last_deployed_at"`
}

type DeploymentConfig struct {
	MaxSurge        int `json:"max_surge"`
	RollingInterval int `json:"rolling_interval"`
	HealthTimeout   int `json:"health_timeout"`

	AutoConfirm         bool `json:"auto_confirm"`
	ConfirmationTimeout int  `json:"confirmation_timeout"`
}

type DeploymentState struct {
	Active  DeploymentColor `json:"active"`
	Standby DeploymentColor `json:"standby"`

	Blue *Environment `json:"blue,omitempty"`

	Green *Environment `json:"green,omitempty"`
}

type Environment struct {
	ContainerIDs []string  `json:"container_ids"`
	ImageID      string    `json:"image_id"`
	DeployedAt   time.Time `json:"deployed_at"`
}

type DeploymentRecord struct {
	ID         string             `json:"id"`
	ImageID    string             `json:"image_id"`
	Strategy   DeploymentStrategy `json:"strategy"`
	DeployedAt time.Time          `json:"deployed_at"`
	Status     string             `json:"status"`
}

type AppRegistry struct {
	Applications []Application `json:"applications"`
}

type Volume struct {
	Name      string    `json:"name"`
	MountPath string    `json:"mount_path"`
	Type      string    `json:"type"`
	Source    string    `json:"source,omitempty"`
	ReadOnly  bool      `json:"read_only,omitempty"`
	Size      string    `json:"size,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type VolumeInfo struct {
	Volume
	DockerName string `json:"docker_name"`
	Driver     string `json:"driver"`
	MountPoint string `json:"mount_point"`
	UsedBy     int    `json:"used_by"`
	ActualSize int64  `json:"actual_size"`
}
