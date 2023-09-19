package service

import (
	"errors"

	"github.com/oodzchen/dproject/config"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
)

type Permission struct {
	Store          *store.Store
	PermissionData *config.PermissionData
	RoleData       config.RoleData
}

func (pm *Permission) InitPermissionTable() error {
	pList, err := pm.Store.Permission.List(1, 999, "all")
	if err != nil {
		return err
	}

	if len(pList) > 0 {
		return nil
	}

	var list []*model.Permission

	if pm.PermissionData == nil || pm.PermissionData.Data == nil {
		return errors.New("permission data is nil")
	}

	for m, v := range pm.PermissionData.Data {
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

	err = pm.Store.Permission.Clear()
	if err != nil {
		return err
	}

	err = pm.Store.Permission.CreateMany(list)
	if err != nil {
		return err
	}

	return nil
}

func (pm *Permission) InitRoleTable() error {
	rList, err := pm.Store.Role.List(1, 999)
	if err != nil {
		return err
	}

	if len(rList) > 0 {
		return nil
	}

	var roleList []*model.Role
	for _, v := range pm.RoleData {
		var pList []*model.Permission
		for _, pFrontId := range v.Permissions {
			pList = append(pList, &model.Permission{
				FrontId: pFrontId,
			})
		}

		roleList = append(roleList, &model.Role{
			FrontId:     v.AdaptId,
			Name:        v.Name,
			Permissions: pList,
		})
	}

	// fmt.Printf("roleList: %#v\n", roleList)

	err = pm.Store.Role.CreateManyWithFrontId(roleList)
	if err != nil {
		return err
	}

	return nil
}
