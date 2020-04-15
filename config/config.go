package config

import (
	"log"

	"github.com/BurntSushi/toml"
)

// Config map data from config.toml
type Config struct {
	Port     string
	Jwtkey   string
	Server   string
	Database string
}

// Read and parse the configuration file
func (c *Config) Read() {
	if _, err := toml.DecodeFile("config.toml", &c); err != nil {
		log.Fatal(err)
	}
}
