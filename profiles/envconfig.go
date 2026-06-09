package profiles

import (
	"encoding/json"
	"os"
)

type EnvConfigEntry struct {
	URLs map[string]string `json:"urls"`
}

type EnvConfig struct {
	Shared       map[string]string         `json:"shared"`
	Environments map[string]EnvConfigEntry `json:"environments"`
}

func ParseEnvConfig(path string) (*EnvConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg EnvConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
