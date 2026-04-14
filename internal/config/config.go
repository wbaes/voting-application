package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig `yaml:"server"`
	Exhibition Exhibition   `yaml:"exhibition"`
	Photos     []Photo      `yaml:"photos"`
}

type ServerConfig struct {
	Port          int    `yaml:"port"`
	AdminPassword string `yaml:"admin_password"`
}

type Exhibition struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

type Photo struct {
	ID       string `yaml:"id"`
	Title    string `yaml:"title"`
	File     string `yaml:"file"`
	Rotation int    `yaml:"rotation"` // clockwise degrees: 0, 90, 180, 270
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}

	// Allow environment variable override for admin password
	if envPass := os.Getenv("ADMIN_PASSWORD"); envPass != "" {
		cfg.Server.AdminPassword = envPass
	}

	if cfg.Server.AdminPassword == "" {
		return nil, fmt.Errorf("admin_password must be set in config or ADMIN_PASSWORD env var")
	}

	if len(cfg.Photos) == 0 {
		return nil, fmt.Errorf("at least one photo must be configured")
	}

	for _, p := range cfg.Photos {
		switch p.Rotation {
		case 0, 90, 180, 270:
			// valid
		default:
			return nil, fmt.Errorf("photo %q: rotation must be 0, 90, 180, or 270 (got %d)", p.ID, p.Rotation)
		}
	}

	return &cfg, nil
}

// PhotoByID returns a photo by its ID, or nil if not found.
func (c *Config) PhotoByID(id string) *Photo {
	for i := range c.Photos {
		if c.Photos[i].ID == id {
			return &c.Photos[i]
		}
	}
	return nil
}
