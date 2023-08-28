package web

import "github.com/go-chi/chi/v5"

type AppState struct {
	router *chi.Mux
}
