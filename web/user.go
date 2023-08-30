package web

import (
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
	"github.com/pkg/errors"
)

type UserResource struct {
	*Renderer
	store *store.Store
}

type userProfile struct {
	UserInfo *model.User
	Posts    []*model.Article
}

func NewUserResource(tmpl *template.Template, store *store.Store, sessStore *sessions.CookieStore, router *chi.Mux) *UserResource {
	return &UserResource{
		&Renderer{tmpl, sessStore, router},
		store,
	}
}

func (ur *UserResource) Routes() http.Handler {
	rt := chi.NewRouter()

	rt.Get("/", ur.List)
	rt.Get("/{userId}", ur.ItemPage)

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

	list, err := ur.store.User.List(page, pageSize)
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

	user, err := ur.store.User.Item(userId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ur.Error("", nil, w, r, http.StatusNotFound)
		} else {
			ur.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		}
		return
	}

	postList, err := ur.store.User.GetPosts(userId)
	if err != nil {
		ur.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		return
	}

	for _, article := range postList {
		article.UpdateDisplayTitle()
		article.GenSummary(200)
	}

	ur.Render(w, r, "user_item", &PageData{
		Title: user.Name,
		Data: &userProfile{
			UserInfo: user,
			Posts:    postList,
		},
		BreadCrumbs: []*BreadCrumb{
			{
				fmt.Sprintf("/users/%d", user.Id),
				user.Name,
			},
		},
	})
}
