package models

import "fmt"

type ProjectConfig struct {
	App        AppConfig           `toml:"app"`
	Build      BuildConfig         `toml:"build"`
	Deployment NapDeploymentConfig `toml:"deployment"`
	Deploy     DeployConfig        `toml:"deploy"`
	Network    NetworkConfig       `toml:"network"`
	Env        map[string]string   `toml:"env"`
	Database   map[string]string   `toml:"database"`
	Volumes    map[string]string   `toml:"volumes"`
	Hooks      HooksConfig         `toml:"hooks"`
	Monitoring MonitoringConfig    `toml:"monitoring"`
	Scaling    ScalingConfig       `toml:"scaling"`
}

type AppConfig struct {
	Name    string `toml:"name"`
	Region  string `toml:"region"`
	Runtime string `toml:"runtime"`
	Env     string `toml:"env"`
}

type BuildConfig struct {
	Dockerfile string   `toml:"dockerfile"`
	Buildpacks bool     `toml:"buildpacks"`
	BuildArgs  []string `toml:"build_args"`
}

type NapDeploymentConfig struct {
	Strategy            string `toml:"strategy"`
	MaxSurge            int    `toml:"max_surge"`
	RollingInterval     int    `toml:"rolling_interval"`
	HealthTimeout       int    `toml:"health_timeout"`
	AutoConfirm         bool   `toml:"auto_confirm"`
	ConfirmationTimeout int    `toml:"confirmation_timeout"`
}

type DeployConfig struct {
	Instances   int               `toml:"instances"`
	Memory      string            `toml:"memory"`
	CPU         float64           `toml:"cpu"`
	Port        int               `toml:"port"`
	AutoScaling bool              `toml:"auto_scaling"`
	HealthCheck HealthCheckConfig `toml:"health_check"`
	Resources   ResourceLimits    `toml:"resources"`
}

type HealthCheckConfig struct {
	Path     string `toml:"path"`
	Interval int    `toml:"interval"`
	Timeout  int    `toml:"timeout"`
	Retries  int    `toml:"retries"`
}

type ResourceLimits struct {
	MemoryLimit string  `toml:"memory_limit"`
	CPULimit    float64 `toml:"cpu_limit"`
}

type NetworkConfig struct {
	SSL          bool   `toml:"ssl"`
	Domain       string `toml:"domain"`
	InternalOnly bool   `toml:"internal_only"`
}

type HooksConfig struct {
	PreBuild   string `toml:"prebuild"`
	PostBuild  string `toml:"postbuild"`
	PreDeploy  string `toml:"predeploy"`
	PostDeploy string `toml:"postdeploy"`
}

type MonitoringConfig struct {
	Metrics       bool `toml:"metrics"`
	LogsRetention int  `toml:"logs_retention"`
	Alerts        bool `toml:"alerts"`
}

type ScalingConfig struct {
	MinInstances    int `toml:"min_instances"`
	MaxInstances    int `toml:"max_instances"`
	CPUThreshold    int `toml:"cpu_threshold"`
	MemoryThreshold int `toml:"memory_threshold"`
	ScaleUpDelay    int `toml:"scale_up_delay"`
	ScaleDownDelay  int `toml:"scale_down_delay"`
}

func ParseMemory(mem string) int {
	if mem == "" {
		return 512
	}

	var value int
	var unit string

	if _, err := fmt.Sscanf(mem, "%d%s", &value, &unit); err != nil {
		return 512
	}

	switch unit {
	case "M", "m", "MB", "mb":
		return value
	case "G", "g", "GB", "gb":
		return value * 1024
	case "K", "k", "KB", "kb":
		return value / 1024
	default:
		// no unit!!! we'll assume you meant metabytes (because who uses bytes anyway)
		return value
	}
}
