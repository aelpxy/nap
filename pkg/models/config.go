package models

type GlobalConfig struct {
	Runtime    RuntimeConfig    `toml:"runtime" json:"runtime"`
	Publishing PublishingConfig `toml:"publishing" json:"publishing"`
}

type RuntimeConfig struct {
	Prefer     string `toml:"prefer" json:"prefer"`
	AutoStart  bool   `toml:"auto_start" json:"auto_start"`
	SocketPath string `toml:"socket_path" json:"socket_path"`
}

type PublishingConfig struct {
	Enabled    bool   `toml:"enabled" json:"enabled"`
	BaseDomain string `toml:"base_domain" json:"base_domain"`
	Email      string `toml:"email" json:"email"`
}
