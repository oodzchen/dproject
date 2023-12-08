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
	PageType        string
}

func NewUserResource(renderer *Renderer) *UserResource {
	return &UserResource{
		renderer,
		renderer.srv.User,
	}
}

func (ur *UserResource) Routes() http.Handler {
	rt := chi.NewRouter()

	rt.Route("/{username}", func(r chi.Router) {
		r.Get("/", ur.ItemPage)

		r.With(mdw.AuthCheck(ur.sessStore), mdw.PermitCheck(
			ur.srv.Permission,
			[]string{"user.update_role"},
			ur,
		)).Group(func(r chi.Router) {
			// r.Get("/ban", ur.BanPage)
			r.Get("/set_role", ur.SetRolePage)
			r.With(mdw.UserLogger(
				ur.uLogger, model.AcTypeManage, model.AcActionSetRole, model.AcModelUser, mdw.ULogLoginedUserId),
			).Post("/set_role", ur.SetRole)
		})

		r.With(mdw.AuthCheck(ur.sessStore), mdw.PermitCheck(
			ur.srv.Permission,
			[]string{"user.update_intro_others"},
			ur,
		)).Group(func(r chi.Router) {
			r.Get("/edit", ur.EditUserProfilePage)
			r.Post("/edit", ur.UpdateUserProfile)
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

	username := r.URL.Query().Get("username")
	roleFrontId := r.URL.Query().Get("role")
	sort := r.URL.Query().Get("sort")

	var oldest bool
	if sort == "oldest" {
		oldest = true
	} else {
		oldest = false
		sort = "latest"
	}

	list, total, err := ur.store.User.List(page, pageSize, oldest, username, roleFrontId)
	if err != nil {
		ur.Error("", err, w, r, http.StatusInternalServerError)
	}

	// total, err := ur.store.User.Count()
	// if err != nil {
	// 	ur.Error("", err, w, r, http.StatusInternalServerError)
	// 	return
	// }

	type UserQueryData struct {
		UserName, Role, Sort   string
		Total, Page, TotalPage int
	}

	type UserListPage struct {
		List []*model.User
		// Total     int
		// CurrPage  int
		// TotalPage int
		// PageSize  int

		Query       *UserQueryData
		RoleOptions []*model.OptionItem
	}

	ur.SavePrevPage(w, r)

	roleList, err := ur.store.Role.List(page, pageSize)
	if err != nil {
		ur.Error("", err, w, r, http.StatusInternalServerError)
	}

	var roleOptions []*model.OptionItem
	for _, item := range roleList {
		roleOptions = append(roleOptions, &model.OptionItem{
			Name:  item.Name,
			Value: item.FrontId,
		})
	}

	ur.Render(w, r, "user_list", &model.PageData{
		Title: "User List",
		Data: &UserListPage{
			List: list,
			// total,
			// page,
			// CeilInt(total, pageSize),
			// pageSize,
			Query: &UserQueryData{
				UserName:  username,
				Role:      roleFrontId,
				Sort:      sort,
				Total:     total,
				Page:      page,
				TotalPage: CeilInt(total, pageSize),
			},
			RoleOptions: roleOptions,
		},
		BreadCrumbs: []*model.BreadCrumb{
			{
				Path: "/users",
				Name: ur.Local("UserList"),
			},
		},
	})
}

func (ur *UserResource) ItemPage(w http.ResponseWriter, r *http.Request) {
	ur.handleItemPage(w, r, "view")
}

func (ur *UserResource) handleItemPage(w http.ResponseWriter, r *http.Request, pageType string) {
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

	// userId, err := strconv.Atoi(chi.URLParam(r, "userId"))
	// if err != nil {
	// 	ur.Error("", errors.WithStack(err), w, r, http.StatusBadRequest)
	// 	return
	// }

	username := chi.URLParam(r, "username")
	if username == "" {
		ur.Error("", errors.New("username is empty"), w, r, http.StatusBadRequest)
		return
	}

	tab := r.URL.Query().Get("tab")
	if tab == "" {
		tab = string(service.UserListAll)
	}

	if !IsLogin(ur.sessStore, w, r) && service.CheckUserTabAuthRequired(service.UserListType(tab)) {
		http.Redirect(w, r, fmt.Sprintf("/users/%s", username), http.StatusFound)
		return
	}

	user, err := ur.store.User.ItemWithUsername(username)
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
		postList, err = ur.userSrv.GetPosts(username, service.UserListType(tab))
	} else {
		if !ur.srv.Permission.PermissionData.Permit("user", "access_activity") {
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
			PageType:        pageType,
			Query: &queryData{
				Total:     total,
				Page:      page,
				TotalPage: CeilInt(total, pageSize),
			},
		},
		BreadCrumbs: []*model.BreadCrumb{
			{
				Path: fmt.Sprintf("/users/%s", user.Name),
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
				Path: fmt.Sprintf("/users/%s", user.Name),
				Name: user.Name,
			},
		},
	})
}

func (ur *UserResource) SetRole(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		ur.Error("", errors.New("username is empty"), w, r, http.StatusBadRequest)
		return
	}

	roleFrontId := r.PostForm.Get("role_front_id")
	comment := r.PostForm.Get("comment")

	// fmt.Println("roleFrontId: ", roleFrontId)
	// role, err := ur.store.Role.Item(int)
	// if !ur.srv.Permission.RoleData.Valid(roleFrontId) {
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

	user, err := ur.store.User.ItemWithUsername(username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ur.Error("", nil, w, r, http.StatusNotFound)
		} else {
			ur.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		}
		return
	}

	if user.RoleFrontId == roleFrontId {
		http.Redirect(w, r, fmt.Sprintf("/users/%s", user.Name), http.StatusFound)
		return
	}

	_, err = ur.store.User.SetRole(user.Id, roleFrontId)
	if err != nil {
		ur.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	if roleFrontId == "banned_user" {
		go func() {
			err := ur.store.User.AddReputation(username, model.RPCTypeBanned, false)
			if err != nil {
				fmt.Println("add reputation error", err)
				return
			}
		}()
	}

	http.Redirect(w, r, fmt.Sprintf("/users/%s", user.Name), http.StatusFound)
}

func (ur *UserResource) SetRolePage(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		ur.Error("", errors.New("username is empty"), w, r, http.StatusBadRequest)
		return
	}

	// roleFrontId := r.URL.Query().Get("role_id")

	user, err := ur.store.User.ItemWithUsername(username)
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

	canSetModerate := ur.srv.Permission.PermissionData.Permit("user", "set_moderator")
	canSetAdmin := ur.srv.Permission.PermissionData.Permit("user", "set_admin")
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
	// 	if ur.srv.Permission.RoleData.Valid(roleFrontId) {
	// 		roleName = ur.srv.Permission.RoleData.Get(config.RoleId(roleFrontId)).Name
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
			// RoleData: ur.srv.Permission.RoleData,
			RoleList: roleList,
		},
		BreadCrumbs: []*model.BreadCrumb{
			{
				Path: fmt.Sprintf("/users/%s", user.Name),
				Name: user.Name,
			},
		},
	})
}

func (ur *UserResource) EditUserProfilePage(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		ur.Error("", errors.New("username is empty"), w, r, http.StatusBadRequest)
		return
	}

	user, err := ur.store.User.ItemWithUsername(username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ur.Error("", nil, w, r, http.StatusNotFound)
		} else {
			ur.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		}
		return
	}

	type pageData struct {
		UserData *model.User
	}

	ur.Render(w, r, "user_profile_form", &model.PageData{
		Title: "Update role of " + user.Name,
		Data: &pageData{
			UserData: user,
		},
		BreadCrumbs: []*model.BreadCrumb{
			{
				Path: fmt.Sprintf("/users/%s", user.Name),
				Name: user.Name,
			},
		},
	})
}

func (ur *UserResource) UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	introduction := r.FormValue("introduction")

	// fmt.Println("introduction:", introduction)

	user := &model.User{
		Introduction: introduction,
	}

	user.Sanitize(ur.sanitizePolicy)

	err := ur.store.User.UpdateIntroduction(username, user.Introduction)
	if err != nil {
		ur.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	oneSess := ur.Session("one", w, r)
	oneSess.Raw.AddFlash(ur.i18nCustom.MustLocalize("AccountSaveSuccess", "", ""))
	oneSess.Raw.Save(r, w)

	http.Redirect(w, r, fmt.Sprintf("/users/%s", username), http.StatusFound)
}
