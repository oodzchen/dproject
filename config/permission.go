package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Permission struct {
	Name    string `yaml:"name"`
	AdaptId string `yaml:"adapt_id"`
	Enabled bool   `yaml:"enabled"`
}

type PermissionMap map[string]map[string]*Permission

type PermissionData struct {
	Modules []string
	Data    PermissionMap
}

func (pd *PermissionData) Permit(module, action string) bool {
	if pd.Data == nil {
		return false
	}

	if m, ok := pd.Data[module]; ok {
		if a, ok := m[action]; ok {
			return a.Enabled
		}
	}
	return false
}

func (pd *PermissionData) Update(permittedIds []string, isSuper bool) *PermissionData {
	idMap := make(map[string]bool)
	for _, id := range permittedIds {
		idMap[id] = true
	}

	for _, v := range pd.Data {
		for _, p := range v {
			if _, ok := idMap[p.AdaptId]; ok || isSuper {
				p.Enabled = true
			} else {
				p.Enabled = false
			}
		}
	}

	return pd
}

func (pd *PermissionData) Valid(module string) bool {
	if pd.Data == nil {
		return false
	}

	_, ok := pd.Data[module]
	return ok
}

func (pd *PermissionData) GetModuleList() []string {
	return pd.Modules
}

func ParsePermissionData(filePath string) (*PermissionData, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	out := PermissionData{}
	err = yaml.Unmarshal(data, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
