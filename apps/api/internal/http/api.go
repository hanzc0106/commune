package http

import (
	"encoding/json"
	stdhttp "net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hanzc0106/commune/apps/api/internal/app"
	"github.com/hanzc0106/commune/apps/api/internal/auth"
)

type API struct {
	service *app.Service
}

func NewAPI(service *app.Service) stdhttp.Handler {
	api := &API{service: service}
	r := chi.NewRouter()
	r.Get("/bootstrap", api.bootstrap)
	r.Post("/init", api.init)
	r.Get("/login-members", api.loginMembers)
	r.Post("/login", api.login)
	r.Get("/session", api.session)
	r.Post("/logout", api.logout)
	return r
}

func (api *API) bootstrap(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	token := sessionTokenFromRequest(r)
	result, err := api.service.Bootstrap(r.Context(), token)
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "bootstrap failed"})
		return
	}
	writeJSON(w, stdhttp.StatusOK, result)
}

func (api *API) init(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var input app.InitializeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	result, token, err := api.service.Initialize(r.Context(), input)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	stdhttp.SetCookie(w, auth.SessionCookie(token, sessionExpiresAt()))
	writeJSON(w, stdhttp.StatusCreated, result)
}

func (api *API) loginMembers(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	members, err := api.service.ListLoginMembers(r.Context())
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "load members failed"})
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]any{"members": members})
}

func (api *API) login(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var input struct {
		MemberID string `json:"memberId"`
		PIN      string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	result, token, err := api.service.Login(r.Context(), app.LoginInput{
		MemberID: input.MemberID,
		PIN:      input.PIN,
	})
	if err != nil {
		writeJSON(w, stdhttp.StatusUnauthorized, map[string]string{"error": "invalid member or PIN"})
		return
	}
	stdhttp.SetCookie(w, auth.SessionCookie(token, sessionExpiresAt()))
	writeJSON(w, stdhttp.StatusOK, result)
}

func (api *API) session(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	token := sessionTokenFromRequest(r)
	if token == "" {
		writeJSON(w, stdhttp.StatusOK, map[string]any{"member": nil})
		return
	}
	session, err := api.service.SessionFromToken(r.Context(), token)
	if err != nil {
		writeJSON(w, stdhttp.StatusOK, map[string]any{"member": nil})
		return
	}
	writeJSON(w, stdhttp.StatusOK, session)
}

func (api *API) logout(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	_ = api.service.Logout(r.Context(), sessionTokenFromRequest(r))
	stdhttp.SetCookie(w, auth.ClearSessionCookie())
	writeJSON(w, stdhttp.StatusOK, map[string]bool{"ok": true})
}

func sessionExpiresAt() time.Time {
	return time.Now().Add(30 * 24 * time.Hour)
}

func sessionTokenFromRequest(r *stdhttp.Request) string {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func writeJSON(w stdhttp.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
