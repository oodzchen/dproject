package web

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oodzchen/dproject/model"
	"github.com/pkg/errors"
)

type ManageResource struct {
	*Renderer
	ur *UserResource
}

func NewManageResource(renderer *Renderer, ur *UserResource) *ManageResource {
	return &ManageResource{
		renderer,
		ur,
	}
}

func (mr *ManageResource) Routes() http.Handler {
	rt := chi.NewRouter()

	rt.Route("/", func(r chi.Router) {
		r.Get("/", mr.PermissionListPage)

		r.Route("/permissions", func(r chi.Router) {
			r.Get("/", mr.PermissionListPage)
			// r.Post("/", mr.PermissionSubmit)
			// r.Get("/new", mr.PermissionCreatePage)
		})

		r.Route("/roles", func(r chi.Router) {
			r.Get("/", mr.RoleListPage)
			r.Post("/", mr.RoleSubmit)
			r.Get("/new", mr.RoleCreatePage)
			r.Get("/{id}/edit", mr.RoleEditPage)
			r.Post("/{id}/edit", mr.RoleEditSubmit)
		})

		// r.Get("/roles", mr.RoleListPage)
		r.Get("/users", mr.ur.List)
	})

	return rt
}

func (mr *ManageResource) PermissionListPage(w http.ResponseWriter, r *http.Request) {
	mr.handlePermissionList(w, r, PermissionPageList)
}

func (mr *ManageResource) PermissionCreatePage(w http.ResponseWriter, r *http.Request) {
	mr.handlePermissionList(w, r, PermissionPageCreate)
}

type PermissionPageType string

const (
	PermissionPageList   PermissionPageType = "list"
	PermissionPageCreate                    = "create"
)

func (mr *ManageResource) handlePermissionList(w http.ResponseWriter, r *http.Request, pageType PermissionPageType) {
	r.ParseForm()

	paramPage := r.Form.Get("page")

	tab := r.Form.Get("tab")
	// fmt.Println("paramPage:", paramPage)
	if !mr.permissionSrv.PermissionData.Valid(tab) {
		tab = "all"
	}

	page, err := strconv.Atoi(paramPage)
	if err != nil {
		// fmt.Printf("page err %v\n", err)
		page = 1
	}

	pageSize, err := strconv.Atoi(r.Form.Get("page_size"))
	if err != nil {
		pageSize = 999
	}

	list, err := mr.store.Permission.List(page, pageSize, tab)
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
	}

	total := len(list)

	type PermissionListPage struct {
		List          []*model.Permission
		Total         int
		CurrPage      int
		TotalPage     int
		PageSize      int
		PageType      string
		ModuleOptions []string
		CurrTab       string
	}

	title := "Permission List"
	breadCrumbs := []*BreadCrumb{
		{
			"/manage/permissions",
			"Permission",
		},
	}

	if pageType == PermissionPageCreate {
		title = "Add Permission"
		breadCrumbs = append(breadCrumbs, &BreadCrumb{
			"",
			"Add Permission",
		})
	}

	if pageType == PermissionPageList {
		mr.SavePrevPage(w, r)
	}

	mr.Render(w, r, "permission_list", &PageData{
		Title: title,
		Data: &PermissionListPage{
			list,
			total,
			page,
			CeilInt(total, pageSize),
			pageSize,
			string(pageType),
			mr.permissionSrv.PermissionData.GetModuleList(),
			tab,
		},
		BreadCrumbs: breadCrumbs,
	})
}

func (mr *ManageResource) PermissionSubmit(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	module := r.Form.Get("module")
	frontId := r.Form.Get("front_id")
	name := r.Form.Get("name")

	permission := &model.Permission{
		Module:  module,
		FrontId: frontId,
		Name:    name,
	}

	permission.TrimSpace()
	// permission.Sanitize()

	moduleValid := mr.permissionSrv.PermissionData.Valid(module)
	if !moduleValid {
		mr.Error("module dose not exist", errors.New("module dose not exist"), w, r, http.StatusBadRequest)
		return
	}

	err := permission.Valid()
	if err != nil {
		mr.Error(err.Error(), err, w, r, http.StatusBadRequest)
		return
	}

	_, err = mr.store.Permission.Create(string(permission.Module), permission.FrontId, permission.Name)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PGErrUniqueViolation {
			mr.Error("the permission already existed", err, w, r, http.StatusBadRequest)
		} else {
			mr.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		}

		return
	}

	mr.Session("one", w, r).Flash("Add permission successfully")
	// http.Redirect(w, r, "/manage/permissions", http.StatusFound)
	mr.ToPrevPage(w, r)
}

func (mr *ManageResource) RoleListPage(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	paramPage := r.Form.Get("page")

	page, err := strconv.Atoi(paramPage)
	if err != nil {
		// fmt.Printf("page err %v\n", err)
		page = 1
	}

	pageSize, err := strconv.Atoi(r.Form.Get("page_size"))
	if err != nil {
		pageSize = 999
	}

	list, err := mr.store.Role.List(page, pageSize)
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
	}

	for _, item := range list {
		item.FormattedPermissions = formatPermissionList(item.Permissions, mr.permissionSrv.PermissionData.GetModuleList())
	}

	total := len(list)

	type RoleListPageData struct {
		List      []*model.Role
		Total     int
		CurrPage  int
		TotalPage int
		PageSize  int
	}

	title := "Role List"
	breadCrumbs := []*BreadCrumb{
		{
			"/manage/roles",
			"Role",
		},
	}

	mr.SavePrevPage(w, r)

	mr.Render(w, r, "role_list", &PageData{
		Title: title,
		Data: &RoleListPageData{
			list,
			total,
			page,
			CeilInt(total, pageSize),
			pageSize,
		},
		BreadCrumbs: breadCrumbs,
	})
}

type RoleFormPageType string

const (
	RoleFormPageAdd  RoleFormPageType = "add"
	RoleFormPageEdit                  = "edit"
)

type RoleFormPageData struct {
	Role                 *model.Role
	RolePermissionIdList []int
	PermissionList       []*model.PermissionListItem
	PageType             RoleFormPageType
}

func (mr *ManageResource) RoleCreatePage(w http.ResponseWriter, r *http.Request) {
	// permissionList, err := mr.store.Permission.List(1, 999, "all")
	// if err != nil {
	// 	mr.Error("", err, w, r, http.StatusInternalServerError)
	// 	return
	// }

	// type RoleCreatePageData struct {
	// 	PermissionList []*model.PermissionListItem
	// }

	filteredPermissionList := mr.getFilteredPermissionList(w, r)

	formattedPermissionList := formatPermissionList(filteredPermissionList, mr.permissionSrv.PermissionData.GetModuleList())

	breadCrumbs := []*BreadCrumb{
		{
			"/manage/roles",
			"Role",
		},
		{
			"",
			"Add Role",
		},
	}

	mr.Render(w, r, "role_form", &PageData{
		Title: "Add Role",
		Data: &RoleFormPageData{
			PermissionList: formattedPermissionList,
			PageType:       RoleFormPageAdd,
		},
		BreadCrumbs: breadCrumbs,
	})
}

func formatPermissionList(rawList []*model.Permission, moduleOptions []string) []*model.PermissionListItem {
	var list []*model.PermissionListItem
	listMap := make(map[string][]*model.Permission)

	for _, item := range rawList {
		if mList, ok := listMap[item.Module]; !ok {
			listMap[item.Module] = []*model.Permission{item}
		} else {
			listMap[item.Module] = append(mList, item)
		}
	}

	for _, module := range moduleOptions {
		if pList, ok := listMap[module]; ok {
			list = append(list, &model.PermissionListItem{
				Module: module,
				List:   pList,
			})
		}
	}

	for _, item := range list {
		sort.Slice(item.List, func(i, j int) bool {
			return rune(item.List[i].Name[0]) < rune(item.List[j].Name[0])
		})
	}

	return list
}

func (mr *ManageResource) RoleSubmit(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	frontId := r.Form.Get("front_id")
	name := r.Form.Get("name")
	permissions := r.Form["permissions"]

	// fmt.Println("permissions: ", permissions)

	role := &model.Role{
		FrontId: frontId,
		Name:    name,
	}

	role.TrimSpace()

	err := role.Valid(false)
	if err != nil {
		mr.Error(err.Error(), err, w, r, http.StatusBadRequest)
		return
	}

	filteredPermissionList := mr.getFilteredPermissionList(w, r)
	filteredPermissionMap := make(map[string]bool)
	for _, pItem := range filteredPermissionList {
		filteredPermissionMap[pItem.FrontId] = true
	}

	var permissionFrontIds []string
	for _, pId := range permissions {
		if _, ok := filteredPermissionMap[pId]; ok {
			permissionFrontIds = append(permissionFrontIds, pId)
		}
	}

	// var permissionIds []int

	// for _, idStr := range permissions {
	// 	id, err := strconv.Atoi(idStr)
	// 	if err == nil {
	// 		permissionIds = append(permissionIds, id)
	// 	}
	// }

	// _, err = mr.store.Role.Create(role.FrontId, role.Name, permissionIds)

	fmt.Println("permissionFrontIds: ", permissionFrontIds)
	_, err = mr.store.Role.CreateWithFrontId(role.FrontId, role.Name, permissionFrontIds)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PGErrUniqueViolation {
			mr.Error("the role is existing", err, w, r, http.StatusBadRequest)
		} else {
			mr.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		}

		return
	}

	mr.Session("one", w, r).Flash("Add role successfully")
	http.Redirect(w, r, "/manage/roles", http.StatusFound)
	// mr.ToPrevPage(w, r)
}

// Fitler permmmsions that not belong to current user
func (mr *ManageResource) getFilteredPermissionList(w http.ResponseWriter, r *http.Request) []*model.Permission {
	permissionList, err := mr.store.Permission.List(1, 999, "all")
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
		return nil
	}

	// userPermittedIdList := mr.Session("one", w, r).GetValue("user_permitted_id_list")
	userPermittedIdList := mr.getUserPermittedFrontIds(r)

	userPermittedIdMap := make(map[string]bool)
	for _, frontId := range userPermittedIdList {
		userPermittedIdMap[frontId] = true
	}

	// if uPList, ok := userPermittedIdList.([]string); ok {
	// 	for _, frontId := range uPList {
	// 		userPermittedIdMap[frontId] = true
	// 	}
	// }

	var filteredPermissionList []*model.Permission
	for _, item := range permissionList {
		if _, ok := userPermittedIdMap[item.FrontId]; ok {
			filteredPermissionList = append(filteredPermissionList, item)
		}
	}
	return filteredPermissionList
}

func (mr *ManageResource) RoleEditPage(w http.ResponseWriter, r *http.Request) {
	roleIdStr := chi.URLParam(r, "id")
	// fmt.Println("roleId: ", roleIdStr)

	roleId, err := strconv.Atoi(roleIdStr)
	if err != nil {
		mr.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	role, err := mr.store.Role.Item(roleId)
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	filteredPermissionList := mr.getFilteredPermissionList(w, r)
	formattedPermissionList := formatPermissionList(filteredPermissionList, mr.permissionSrv.PermissionData.GetModuleList())

	var rolePermissionIdList []int
	if role.Permissions != nil {
		for _, item := range role.Permissions {
			rolePermissionIdList = append(rolePermissionIdList, item.Id)
		}
	}

	breadCrumbs := []*BreadCrumb{
		{
			"/manage/roles",
			"Role",
		},
		{
			"",
			"Edit Role",
		},
	}

	mr.Render(w, r, "role_form", &PageData{
		Title: "Edit Role",
		Data: &RoleFormPageData{
			Role:                 role,
			RolePermissionIdList: rolePermissionIdList,
			PermissionList:       formattedPermissionList,
			PageType:             RoleFormPageEdit,
		},
		BreadCrumbs: breadCrumbs,
	})
}

func (mr *ManageResource) RoleEditSubmit(w http.ResponseWriter, r *http.Request) {
	roleIdStr := chi.URLParam(r, "id")
	// fmt.Println("roleId: ", roleIdStr)

	roleId, err := strconv.Atoi(roleIdStr)
	if err != nil {
		mr.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	// isDefault := r.Form.Get("is_default")
	role, err := mr.store.Role.Item(roleId)
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	if role.IsDefault {
		mr.Error("", nil, w, r, http.StatusForbidden)
		return
	}

	name := r.Form.Get("name")
	permissions := r.Form["permissions"]

	// fmt.Println("permissions: ", permissions)

	role = &model.Role{
		Id:   roleId,
		Name: name,
	}

	role.TrimSpace()

	err = role.Valid(true)
	if err != nil {
		mr.Error(err.Error(), err, w, r, http.StatusBadRequest)
		return
	}

	// var permissionIds []int

	// for _, idStr := range permissions {
	// 	id, err := strconv.Atoi(idStr)
	// 	if err == nil {
	// 		permissionIds = append(permissionIds, id)
	// 	}
	// }

	// _, err = mr.store.Role.Update(role.Id, role.Name, permissionIds)

	filteredPermissionList := mr.getFilteredPermissionList(w, r)
	filteredPermissionMap := make(map[string]bool)
	for _, pItem := range filteredPermissionList {
		filteredPermissionMap[pItem.FrontId] = true
	}

	var permissionFrontIds []string
	for _, pId := range permissions {
		if _, ok := filteredPermissionMap[pId]; ok {
			permissionFrontIds = append(permissionFrontIds, pId)
		}
	}

	_, err = mr.store.Role.UpdateWithFrontId(role.Id, role.Name, permissionFrontIds)

	if err != nil {
		mr.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)

		return
	}

	mr.Session("one", w, r).Flash("Update role successfully")

	http.Redirect(w, r, "/manage/roles", http.StatusFound)

}
