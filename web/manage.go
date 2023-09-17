package web

import (
	"net/http"
	"strconv"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
	"github.com/pkg/errors"
)

type ManageResource struct {
	*Renderer
	ur *UserResource
}

func NewManageResource(tmpl *template.Template, store *store.Store, sessStore *sessions.CookieStore, router *chi.Mux, ur *UserResource) *ManageResource {
	return &ManageResource{
		&Renderer{
			tmpl,
			sessStore,
			router,
			store,
		},
		ur,
	}
}

func (mr *ManageResource) Routes() http.Handler {
	rt := chi.NewRouter()

	rt.Route("/", func(r chi.Router) {
		r.Get("/", mr.PermissionListPage)

		r.Route("/permissions", func(r chi.Router) {
			r.Get("/", mr.PermissionListPage)
			r.Post("/", mr.PermissionSubmit)
			r.Get("/new", mr.PermissionCreatePage)
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
	if !model.ValidPermissionModule(tab) {
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

	total, err := mr.store.User.Count()
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	type PermissionListPage struct {
		List          []*model.Permission
		Total         int
		CurrPage      int
		TotalPage     int
		PageSize      int
		PageType      string
		ModuleOptions []model.PermissionModule
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
			model.GetPermissionModuleOptions(),
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
		Module:  model.PermissionModule(module),
		FrontId: frontId,
		Name:    name,
	}

	permission.TrimSpace()
	// permission.Sanitize()

	err := permission.Valid()
	if err != nil {
		mr.Error(err.Error(), err, w, r, http.StatusBadRequest)
		return
	}

	_, err = mr.store.Permission.Create(string(permission.Module), permission.FrontId, permission.Name)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PGErrUniqueViolation {
			mr.Error("the permission is existing", err, w, r, http.StatusBadRequest)
		} else {
			mr.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		}

		return
	}

	mr.Session("one", w, r).Flash("Add permission successfully")
	// http.Redirect(w, r, "/manage/permissions", http.StatusFound)
	mr.ToPrevPage(w, r)
}
