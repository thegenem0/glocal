package config

import "github.com/spf13/viper"

type Config struct {
	Server     ServerConfig               `mapstructure:"server"`
	Services   map[string]ServiceConfig   `mapstructure:"services"`
	Containers map[string]ContainerConfig `mapstructure:"containers"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type ServiceConfig struct {
	Enabled   bool           `mapstructure:"enabled"`
	Container string         `mapstructure:"container"`
	Config    map[string]any `mapstructure:"config"`
}

type ContainerConfig struct {
	Image       string            `mapstructure:"image"`
	Ports       []int             `mapstructure:"ports"`
	Environment map[string]string `mapstructure:"environment"`
	WaitFor     WaitForConfig     `mapstructure:"wait_for"`
}

type WaitForConfig struct {
	Port int    `mapstructure:"port"`
	Path string `mapstructure:"path"`
}

func LoadConfig(path string) (*Config, error) {
	vi := viper.GetViper()
	vi.SetConfigFile(path)
	vi.SetConfigType("yaml")

	vi.SetDefault("server.port", 8080)
	vi.SetDefault("server.host", "localhost")

	if err := vi.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := vi.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
