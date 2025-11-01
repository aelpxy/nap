package docker

import "time"

const (
	ImagePullTimeout    = 10 * time.Minute
	ImageBuildTimeout   = 15 * time.Minute
	ContainerOpTimeout  = 30 * time.Second
	HealthCheckTimeout  = 2 * time.Minute
	NetworkOpTimeout    = 30 * time.Second
)
