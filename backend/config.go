package backend

import (
	"os"

	"github.com/google/uuid"
	"github.com/pelletier/go-toml"
)

type ServerConfig struct {
	ID       uuid.UUID
	Nickname string
	Hostname string
	Username string
	Default  bool
}

type Config struct {
	Servers []*ServerConfig
}

func DefaultConfig() *Config {
	return &Config{}
}

func ReadConfigFile(filepath string) (*Config, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	c := &Config{}
	if err := toml.NewDecoder(f).Decode(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) GetDefaultServer() *ServerConfig {
	for _, s := range c.Servers {
		if s.Default {
			return s
		}
	}
	if len(c.Servers) > 0 {
		return c.Servers[0]
	}
	return nil
}

func (c *Config) SetDefaultServer(serverID uuid.UUID) {
	var found bool
	for _, s := range c.Servers {
		f := s.ID == serverID
		if f {
			found = true
		}
		s.Default = f
	}
	if !found && len(c.Servers) > 0 {
		c.Servers[0].Default = true
	}
}

func (c *Config) AddServer(nickname, hostname, username string) *ServerConfig {
	s := &ServerConfig{
		ID:       uuid.New(),
		Nickname: nickname,
		Hostname: hostname,
		Username: username,
	}
	c.Servers = append(c.Servers, s)
	return s
}

func (c *Config) WriteConfigFile(filepath string) error {
	b, err := toml.Marshal(c)
	if err != nil {
		return err
	}
	os.WriteFile(filepath, b, 0644)

	return nil
}
