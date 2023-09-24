package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/service"
	"github.com/pkg/errors"
)

type UserResource struct {
	*Renderer
	userSrv *service.User
	// store *store.Store
}

type userProfile struct {
	UserInfo        *model.User
	Posts           []*model.Article
	CurrTab         service.UserListType
	PermissionNames []string
}

func NewUserResource(renderer *Renderer) *UserResource {
	return &UserResource{
		renderer,
		&service.User{
			Store: renderer.store,
		},
	}
}

func (ur *UserResource) Routes() http.Handler {
	rt := chi.NewRouter()

	// rt.Get("/", ur.List)

	rt.Route("/{userId}", func(r chi.Router) {
		r.Get("/", ur.ItemPage)
		r.Get("/ban", ur.BanPage)
		r.Post("/set_role", ur.SetRole)
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

	ur.Render(w, r, "user_list", &PageData{
		Title: "User List",
		Data: &UserListPage{
			list,
			total,
			page,
			CeilInt(total, pageSize),
			pageSize,
		},
		BreadCrumbs: []*BreadCrumb{
			{
				"/users",
				"User List",
			},
		},
	})
}

func (ur *UserResource) ItemPage(w http.ResponseWriter, r *http.Request) {
	// sess, err := ur.sessStore.Get(r, "one")
	// if err != nil{
	// 	ur.Error("", err, w, r, http.StatusInternalServerError)
	// 	return
	// }

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
		if errors.Is(err, pgx.ErrNoRows) {
			ur.Error("", nil, w, r, http.StatusNotFound)
		} else {
			ur.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		}
		return
	}

	postList, err := ur.userSrv.GetPosts(userId, service.UserListType(tab))
	if err != nil {
		ur.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		return
	}

	for _, article := range postList {
		article.UpdateDisplayTitle()
		article.GenSummary(200)
	}

	var permissionNames []string
	for _, item := range user.Permissions {
		permissionNames = append(permissionNames, item.Name)
	}

	ur.Render(w, r, "user_item", &PageData{
		Title: user.Name,
		Data: &userProfile{
			UserInfo:        user,
			Posts:           postList,
			CurrTab:         service.UserListType(tab),
			PermissionNames: permissionNames,
		},
		BreadCrumbs: []*BreadCrumb{
			{
				fmt.Sprintf("/users/%d", user.Id),
				user.Name,
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
		ur.Session("one", w, r).Flash("already banned")
		http.Redirect(w, r, fmt.Sprintf("/users/%d", user.Id), http.StatusFound)
		return
	}

	type pageData struct {
		UserData *model.User
	}

	ur.Render(w, r, "user_role_form", &PageData{
		Title: "Confirm to ban " + user.Name,
		Data: &pageData{
			UserData: user,
		},
		BreadCrumbs: []*BreadCrumb{
			{
				fmt.Sprintf("/users/%d", user.Id),
				user.Name,
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

	fmt.Println("roleFrontId: ", roleFrontId)
	if !ur.permissionSrv.RoleData.Valid(roleFrontId) {
		ur.Error("", errors.New("role front id dose not exist"), w, r, http.StatusBadRequest)
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

	//

	http.Redirect(w, r, fmt.Sprintf("/users/%d", user.Id), http.StatusFound)
}
