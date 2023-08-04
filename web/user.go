package web

import (
	"net/http"
	"strconv"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/utils"
	"github.com/pkg/errors"
)

type UserResource struct {
	Renderer
	store     *store.Store
	sessStore *sessions.CookieStore
}

type userProfile struct {
	UserInfo *model.User
	Posts    []*model.Article
}

func NewUserResource(tmpl *template.Template, store *store.Store, sessStore *sessions.CookieStore) *UserResource {
	return &UserResource{
		Renderer{tmpl, sessStore},
		store,
		sessStore,
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
	// 	utils.HttpError("", err, w, http.StatusInternalServerError)
	// 	return
	// }

	userId, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		utils.HttpError("user id is required", errors.WithStack(err), w, http.StatusBadRequest)
		return
	}

	user, err := ur.store.User.Item(userId)
	if err != nil {
		utils.HttpError("", errors.WithStack(err), w, http.StatusInternalServerError)
		return
	}

	postList, err := ur.store.User.GetPosts(userId)
	if err != nil {
		utils.HttpError("", errors.WithStack(err), w, http.StatusInternalServerError)
		return
	}

	for _, article := range postList {
		if article.ReplyTo > 0 {
			article.Title = "Re: " + article.ReplyToTitle
		}
	}

	ur.Render(w, r, "user_item", &PageData{Title: user.Name, Data: &userProfile{
		UserInfo: user,
		Posts:    postList,
	}})
}
