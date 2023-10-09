package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	mdw "github.com/oodzchen/dproject/middleware"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/service"
	"github.com/pkg/errors"
)

type UserResource struct {
	*Renderer
	userSrv *service.User
	// store *store.Store
}

type queryData struct {
	Total, Page, TotalPage int
}

type userProfile struct {
	UserInfo        *model.User
	Posts           []*model.Article
	CurrTab         service.UserListType
	PermissionNames []string
	Activities      []*model.Activity
	Query           *queryData
}

func NewUserResource(renderer *Renderer) *UserResource {
	return &UserResource{
		renderer,
		&service.User{
			Store:         renderer.store,
			SantizePolicy: renderer.sanitizePolicy,
		},
	}
}

func (ur *UserResource) Routes() http.Handler {
	rt := chi.NewRouter()

	rt.Route("/{userId}", func(r chi.Router) {
		r.Get("/", ur.ItemPage)

		r.With(mdw.AuthCheck(ur.sessStore), mdw.PermitCheck(ur.permissionSrv, []string{
			"user.update_role",
		}, ur)).Group(func(r chi.Router) {
			// r.Get("/ban", ur.BanPage)
			r.Get("/set_role", ur.SetRolePage)
			r.With(mdw.UserLogger(
				ur.uLogger, model.AcTypeManage, model.AcActionSetRole, model.AcModelUser, mdw.ULogLoginedUserId),
			).Post("/set_role", ur.SetRole)
		})
	})

	return rt
}

func (ur *UserResource) List(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	paramPage := r.Form.Get("page")
	// fmt.Println("paramPage:", paramPage)
	page, err := strconv.Atoi(paramPage)
	if err != nil {
		// fmt.Printf("page err %v\n", err)
		page = 1
	}

	pageSize, err := strconv.Atoi(r.Form.Get("page_size"))
	if err != nil {
		pageSize = 100
	}

	list, err := ur.store.User.List(page, pageSize, false)
	if err != nil {
		ur.Error("", err, w, r, http.StatusInternalServerError)
	}

	total, err := ur.store.User.Count()
	if err != nil {
		ur.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	type UserListPage struct {
		List      []*model.User
		Total     int
		CurrPage  int
		TotalPage int
		PageSize  int
	}

	ur.SavePrevPage(w, r)

	ur.Render(w, r, "user_list", &model.PageData{
		Title: "User List",
		Data: &UserListPage{
			list,
			total,
			page,
			CeilInt(total, pageSize),
			pageSize,
		},
		BreadCrumbs: []*model.BreadCrumb{
			{
				Path: "/users",
				Name: "User List",
			},
		},
	})
}

func (ur *UserResource) ItemPage(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	if page < DefaultPage {
		page = DefaultPage
	}

	if pageSize < DefaultPageSize {
		pageSize = DefaultPageSize
	}

	userId, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		ur.Error("", errors.WithStack(err), w, r, http.StatusBadRequest)
		return
	}

	tab := r.URL.Query().Get("tab")
	if tab == "" {
		tab = string(service.UserListAll)
	}

	if !IsLogin(ur.sessStore, w, r) && service.CheckUserTabAuthRequired(service.UserListType(tab)) {
		http.Redirect(w, r, fmt.Sprintf("/users/%d", userId), http.StatusFound)
		return
	}

	user, err := ur.store.User.Item(userId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, model.AppErrUserNotExist) {
			ur.Error("", nil, w, r, http.StatusNotFound)
		} else {
			ur.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		}
		return
	}

	var postList []*model.Article
	var activityList []*model.Activity
	var total int
	if tab != "activity" {
		postList, err = ur.userSrv.GetPosts(userId, service.UserListType(tab))
	} else {
		if !ur.permissionSrv.PermissionData.Permit("user", "access_activity") {
			ur.Error("", nil, w, r, http.StatusForbidden)
			return
		}
		activityList, total, err = ur.store.Activity.List(user.Id, "", "", "", page, pageSize)
	}
	if err != nil {
		ur.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		return
	}

	for _, article := range postList {
		article.UpdateDisplayTitle()
		article.GenSummary(200)
	}

	for _, activity := range activityList {
		activity.Format(ur.i18nCustom)
	}

	var permissionNames []string
	for _, item := range user.Permissions {
		permissionNames = append(permissionNames, item.Name)
	}

	ur.Render(w, r, "user_item", &model.PageData{
		Title: user.Name,
		Data: &userProfile{
			UserInfo:        user,
			Posts:           postList,
			CurrTab:         service.UserListType(tab),
			PermissionNames: permissionNames,
			Activities:      activityList,
			Query: &queryData{
				Total:     total,
				Page:      page,
				TotalPage: CeilInt(total, pageSize),
			},
		},
		BreadCrumbs: []*model.BreadCrumb{
			{
				Path: fmt.Sprintf("/users/%d", user.Id),
				Name: user.Name,
			},
		},
	})
}

func (ur *UserResource) BanPage(w http.ResponseWriter, r *http.Request) {
	userId, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		ur.Error("", errors.WithStack(err), w, r, http.StatusBadRequest)
		return
	}

	user, err := ur.store.User.Item(userId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ur.Error("", nil, w, r, http.StatusNotFound)
		} else {
			ur.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		}
		return
	}

	if user.RoleFrontId == model.DefaultUserRoleBanned {
		ur.Session("one", w, r).Flash("Already banned")
		http.Redirect(w, r, fmt.Sprintf("/users/%d", user.Id), http.StatusFound)
		return
	}

	type pageData struct {
		UserData *model.User
	}

	ur.Render(w, r, "user_role_form", &model.PageData{
		Title: "Confirm to ban " + user.Name,
		Data: &pageData{
			UserData: user,
		},
		BreadCrumbs: []*model.BreadCrumb{
			{
				Path: fmt.Sprintf("/users/%d", user.Id),
				Name: user.Name,
			},
		},
	})
}

func (ur *UserResource) SetRole(w http.ResponseWriter, r *http.Request) {
	userId, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		ur.Error("", errors.WithStack(err), w, r, http.StatusBadRequest)
		return
	}

	roleFrontId := r.PostForm.Get("role_front_id")
	comment := r.PostForm.Get("comment")

	// fmt.Println("roleFrontId: ", roleFrontId)
	// role, err := ur.store.Role.Item(int)
	// if !ur.permissionSrv.RoleData.Valid(roleFrontId) {
	// 	ur.Error("", errors.New("role front id dose not exist"), w, r, http.StatusBadRequest)
	// 	return
	// }

	if strings.TrimSpace(roleFrontId) == "" {
		ur.Error("", errors.New("role id is required"), w, r, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(comment) == "" {
		ur.Error("", errors.New("reason is required"), w, r, http.StatusBadRequest)
		return
	}

	user, err := ur.store.User.Item(userId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ur.Error("", nil, w, r, http.StatusNotFound)
		} else {
			ur.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		}
		return
	}

	if user.RoleFrontId == roleFrontId {
		http.Redirect(w, r, fmt.Sprintf("/users/%d", user.Id), http.StatusFound)
		return
	}

	_, err = ur.store.User.SetRole(user.Id, roleFrontId)
	if err != nil {
		ur.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/users/%d", user.Id), http.StatusFound)
}

func (ur *UserResource) SetRolePage(w http.ResponseWriter, r *http.Request) {
	userId, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		ur.Error("", errors.WithStack(err), w, r, http.StatusBadRequest)
		return
	}

	// roleFrontId := r.URL.Query().Get("role_id")

	user, err := ur.store.User.Item(userId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ur.Error("", nil, w, r, http.StatusNotFound)
		} else {
			ur.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		}
		return
	}

	wholeRoleList, err := ur.store.Role.List(1, 999)
	if err != nil {
		ur.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	var roleList []*model.Role

	canSetModerate := ur.permissionSrv.PermissionData.Permit("user", "set_moderator")
	canSetAdmin := ur.permissionSrv.PermissionData.Permit("user", "set_admin")
	for _, item := range wholeRoleList {
		if item.FrontId == "moderator" && !canSetModerate {
			continue
		}

		if item.FrontId == "admin" && !canSetAdmin {
			continue
		}

		roleList = append(roleList, item)
	}

	// fmt.Println("roleList: ", roleList)

	// if user.RoleFrontId == model.DefaultUserRoleBanned {
	// 	ur.Session("one", w, r).Flash("Already banned")
	// 	http.Redirect(w, r, fmt.Sprintf("/users/%d", user.Id), http.StatusFound)
	// 	return
	// }

	// var roleName string

	// if strings.TrimSpace(roleFrontId) != "" {
	// 	if ur.permissionSrv.RoleData.Valid(roleFrontId) {
	// 		roleName = ur.permissionSrv.RoleData.Get(config.RoleId(roleFrontId)).Name
	// 	} else {
	// 		ur.Error("", errors.New("role id dose not exist"), w, r, http.StatusBadRequest)
	// 		return
	// 	}
	// }

	type pageData struct {
		UserData *model.User
		// RoleFrontId string
		// RoleName    string
		// RoleData *config.RoleData
		RoleList []*model.Role
	}

	ur.Render(w, r, "user_role_form", &model.PageData{
		Title: "Update role of " + user.Name,
		Data: &pageData{
			UserData: user,
			// RoleFrontId: roleFrontId,
			// RoleName:    roleName,
			// RoleData: ur.permissionSrv.RoleData,
			RoleList: roleList,
		},
		BreadCrumbs: []*model.BreadCrumb{
			{
				Path: fmt.Sprintf("/users/%d", user.Id),
				Name: user.Name,
			},
		},
	})
}
