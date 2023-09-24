package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type RoleItem struct {
	Name        string   `yaml:"name"`
	AdaptId     string   `yaml:"adapt_id"`
	Permissions []string `yaml:"permissions,flow"`
}

type RoleData map[string]*RoleItem

func (rd RoleData) Get(roleId string) *RoleItem {
	return rd[roleId]
}

func ParseRoleData(filePath string) (*RoleData, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	out := RoleData{}
	err = yaml.Unmarshal(data, &out)
	if err != nil {
		return nil, err
	}

	return &out, nil
}
