package service

import (
	"errors"

	"github.com/oodzchen/dproject/config"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
)

type Permission struct {
	Store          *store.Store
	PermissionData config.PermissionMap
}

func (pm *Permission) InitPermissionTable() error {
	var list []*model.Permission

	for m, v := range pm.PermissionData {
		for _, p := range v {
			list = append(list, &model.Permission{
				Module:  m,
				FrontId: p.AdaptId,
				Name:    p.Name,
			})
		}
	}

	if len(list) == 0 {
		return errors.New("no data")
	}

	err := pm.Store.Permission.Clear()
	if err != nil {
		return err
	}

	err = pm.Store.Permission.CreateMany(list)
	if err != nil {
		return err
	}

	return nil
}
