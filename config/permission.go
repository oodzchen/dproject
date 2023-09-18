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

func (pm PermissionMap) Permit(module, action string) bool {
	if m, ok := pm[module]; ok {
		if a, ok := m[action]; ok {
			return a.Enabled
		}
	}
	return false
}

func (pm PermissionMap) Update(permittedIds []string) PermissionMap {
	idMap := make(map[string]bool)
	for _, id := range permittedIds {
		idMap[id] = true
	}

	for _, v := range pm {
		for _, p := range v {
			if _, ok := idMap[p.AdaptId]; ok {
				p.Enabled = true
			} else {
				p.Enabled = false
			}
		}
	}

	return pm
}

func (pm PermissionMap) Valid(module string) bool {
	_, ok := pm[module]
	return ok
}

func (pm PermissionMap) GetModuleList() []string {
	var list []string
	for module := range pm {
		list = append(list, module)
	}
	return list
}

func ParsePermissionData(filePath string) (PermissionMap, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	out := PermissionMap{}
	err = yaml.Unmarshal(data, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
