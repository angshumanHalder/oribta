package profiles

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type RewriteRule struct {
	From string
	To   string
}

type Environment struct {
	Name         string
	Headers      map[string]string
	RewriteRules []RewriteRule
}

type ProfileStore struct {
	Environments []Environment `json:"environments"`
	Active       string        `json:"active"`
	PACDomains   []string      `json:"pac_domains"`
	path         string        `json:"-"`
}

func Load(path string) (*ProfileStore, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ProfileStore{
				path: path,
			}, nil
		}
		return nil, err
	}
	var store ProfileStore
	if err := json.Unmarshal(file, &store); err != nil {
		return nil, fmt.Errorf("Invalid config: %s", err)
	}
	store.path = path
	return &store, nil
}

func (s *ProfileStore) Save() error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(s.path), 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(s.path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (s *ProfileStore) ActiveEnv() *Environment {
	for i := 0; i < len(s.Environments); i++ {
		if s.Environments[i].Name == s.Active {
			return &s.Environments[i]
		}
	}
	return nil
}

func (s *ProfileStore) SetActiveEnv(name string) error {
	for _, env := range s.Environments {
		if env.Name == name {
			s.Active = name
			return nil
		}
	}
	return fmt.Errorf("environment %q not found", name)
}
