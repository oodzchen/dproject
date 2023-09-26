package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type RoleType string

const (
	RoleTypeGroup string = "group"
	RoleTypeRole         = "role"
)

type RoleItem struct {
	Name        string      `yaml:"name"`
	AdaptId     string      `yaml:"adapt_id"`
	Permissions []string    `yaml:"permissions,flow"`
	Type        RoleType    `yaml:"type"`
	SubRoles    []*RoleItem `yaml:"sub_roles,flow"`
}

type RoleId string
type RoleDataMap map[RoleId]*RoleItem
type RoleData struct {
	RoleIdList []RoleId `yaml:"role_id_list,flow"`
	Data       RoleDataMap
}

func (rd RoleData) Get(roleId RoleId) *RoleItem {
	return rd.Data[roleId]
}

func (rd RoleData) Valid(roleId string) bool {
	_, ok := rd.Data[RoleId(roleId)]
	return ok
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
