package web

import (
	"net/http"
	"strconv"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
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

func NewUserResource(tmpl *template.Template, store *store.Store, sessStore *sessions.CookieStore) *UserResource {
	return &UserResource{
		&Renderer{tmpl, sessStore},
		store,
	}
}

func (ur *UserResource) Routes() http.Handler {
	rt := chi.NewRouter()

	rt.Get("/{userId}", ur.ItemPage)

	return rt
}

func (ur *UserResource) ItemPage(w http.ResponseWriter, r *http.Request) {
	// sess, err := ur.sessStore.Get(r, "one-cookie")
	// if err != nil{
	// 	ur.Error("", err, w, r, http.StatusInternalServerError)
	// 	return
	// }

	userId, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		ur.Error("user id is required", errors.WithStack(err), w, r, http.StatusBadRequest)
		return
	}

	user, err := ur.store.User.Item(userId)
	if err != nil {
		ur.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		return
	}

	postList, err := ur.store.User.GetPosts(userId)
	if err != nil {
		ur.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		return
	}

	for _, article := range postList {
		article.UpdateDisplayTitle()
	}

	ur.Render(w, r, "user_item", &PageData{Title: user.Name, Data: &userProfile{
		UserInfo: user,
		Posts:    postList,
	}})
}
