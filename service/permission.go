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
	RoleData       *config.RoleData
	loginedUser    *model.User
}

func (pm *Permission) SetLoginedUser(u *model.User) {
	pm.loginedUser = u

	var permittedIdList []string
	for _, item := range u.Permissions {
		permittedIdList = append(permittedIdList, item.FrontId)
	}

	pm.PermissionData.Update(permittedIdList, u.Super)
}

func (pm *Permission) InitPermissionTable() error {
	// fmt.Println("permission store: ", pm.Store)
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
	roleData := *pm.RoleData

	for _, v := range roleData.Data {
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

func (pm *Permission) InitUserRoleTable() error {
	uList, _, err := pm.Store.User.List(1, 999, true, "", "")
	if err != nil {
		return err
	}

	if len(uList) == 0 {
		return nil
	}

	// fmt.Println("uList[0].RoleFrontId: ", uList[0].RoleFrontId)

	if uList[0].RoleFrontId != "" {
		return nil
	}

	for _, item := range uList {
		item.RoleFrontId = string(model.DefaultUserRoleCommon)
	}

	err = pm.Store.User.SetRoleManyWithFrontId(uList)
	if err != nil {
		return err
	}

	return nil
}

func (pm *Permission) ResetPermissionData() error {
	data, err := pm.PermissionData.DefaultData()

	if err != nil {
		return err
	}

	pm.PermissionData = data

	return nil
}

func (pm *Permission) Permit(module string, action string) bool {
	if pm.PermissionData == nil {
		return false
	}
	return pm.PermissionData.Permit(module, action)
}
