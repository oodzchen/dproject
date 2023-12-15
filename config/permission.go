package config

import (
	"fmt"
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
	// EnabledFrondIdList []string
	currentConfigFile string
}

// func (pd *PermissionData) Permit(module, action string) bool {
// 	if pd.Data == nil {
// 		return false
// 	}

// 	if m, ok := pd.Data[module]; ok {
// 		if a, ok := m[action]; ok {
// 			return a.Enabled
// 		}
// 	}
// 	return false
// }

// func (pd *PermissionData) Update(permittedIds []string, isSuper bool) *PermissionData {
// 	idMap := make(map[string]bool)
// 	for _, id := range permittedIds {
// 		idMap[id] = true
// 	}

// 	for _, v := range pd.Data {
// 		for _, p := range v {
// 			if _, ok := idMap[p.AdaptId]; ok || isSuper {
// 				p.Enabled = true
// 			} else {
// 				p.Enabled = false
// 			}
// 		}
// 	}

// 	pd.UpdateEnabledIdList()

// 	return pd
// }

// AdaptId is to associate with backend return permission ID, witch is for the separation of frontend and backend
// In frontend we only use module and action name
func (pd *PermissionData) GetEnabledFrontIdList(permittedIds []string, isSuper bool) []string {
	var idList []string
	idMap := make(map[string]bool)
	for _, id := range permittedIds {
		idMap[id] = true
	}

	for module, moduleVal := range pd.Data {
		for action, actionVal := range moduleVal {
			if _, ok := idMap[actionVal.AdaptId]; ok || isSuper {
				idList = append(idList, fmt.Sprintf("%s.%s", module, action))
			}
		}
	}

	return idList
}

func (pd *PermissionData) GetDefaultEnabledFrontIdList() []string {
	var idList []string
	for _, v := range pd.Data {
		for _, p := range v {
			if p.Enabled {
				idList = append(idList, p.AdaptId)
			}
		}
	}

	return idList
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

func (pd *PermissionData) DefaultData() (*PermissionData, error) {
	return ParsePermissionData(pd.currentConfigFile)
}

// func (pd *PermissionData) UpdateEnabledIdList() {
// 	var idList []string
// 	for _, v1 := range pd.Data {
// 		for _, v2 := range v1 {
// 			if v2.Enabled {
// 				idList = append(idList, v2.AdaptId)
// 			}
// 		}
// 	}

// 	pd.EnabledFrondIdList = idList
// }

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

	out.currentConfigFile = filePath

	// out.UpdateEnabledIdList()

	return &out, nil
}
