package http

import (
	"encoding/json"
	stdhttp "net/http"
	"strings"
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
	r.Get("/members", api.members)
	r.Post("/members", api.createMember)
	r.Post("/members/{id}/disable", api.disableMember)
	r.Post("/members/{id}/reset-pin", api.resetMemberPIN)
	r.Post("/me/change-pin", api.changeOwnPIN)
	r.Get("/categories", api.categories)
	r.Post("/categories", api.createCategory)
	r.Patch("/categories/{id}", api.updateCategory)
	r.Post("/categories/{id}/disable", api.disableCategory)
	r.Get("/transactions", api.transactions)
	r.Post("/transactions", api.createTransaction)
	r.Patch("/transactions/{id}", api.updateTransaction)
	r.Delete("/transactions/{id}", api.deleteTransaction)
	r.Get("/overview/monthly", api.monthlyOverview)
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

func (api *API) members(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	member, ok := api.requireSession(w, r)
	if !ok {
		return
	}
	members, err := api.service.ListMembers(r.Context(), member)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]any{"members": members})
}

func (api *API) createMember(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	member, ok := api.requireSession(w, r)
	if !ok {
		return
	}
	var input app.CreateMemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	created, err := api.service.CreateMember(r.Context(), member, input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, stdhttp.StatusCreated, created)
}

func (api *API) disableMember(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	member, ok := api.requireSession(w, r)
	if !ok {
		return
	}
	disabled, err := api.service.DisableMember(r.Context(), member, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, stdhttp.StatusOK, disabled)
}

func (api *API) resetMemberPIN(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	member, ok := api.requireSession(w, r)
	if !ok {
		return
	}
	var input app.ResetMemberPINInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if err := api.service.ResetMemberPIN(r.Context(), member, chi.URLParam(r, "id"), input); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]bool{"ok": true})
}

func (api *API) changeOwnPIN(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	member, ok := api.requireSession(w, r)
	if !ok {
		return
	}
	var input app.ChangeOwnPINInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if err := api.service.ChangeOwnPIN(r.Context(), member, input); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]bool{"ok": true})
}

func (api *API) categories(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if _, ok := api.requireSession(w, r); !ok {
		return
	}
	categories, err := api.service.ListCategories(r.Context())
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": "load categories failed"})
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]any{"categories": categories})
}

func (api *API) createCategory(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	member, ok := api.requireSession(w, r)
	if !ok {
		return
	}
	var input app.CreateCategoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	category, err := api.service.CreateCategory(r.Context(), member, input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, stdhttp.StatusCreated, category)
}

func (api *API) updateCategory(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	member, ok := api.requireSession(w, r)
	if !ok {
		return
	}
	var input app.UpdateCategoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	category, err := api.service.UpdateCategory(r.Context(), member, chi.URLParam(r, "id"), input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, stdhttp.StatusOK, category)
}

func (api *API) disableCategory(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	member, ok := api.requireSession(w, r)
	if !ok {
		return
	}
	category, err := api.service.DisableCategory(r.Context(), member, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, stdhttp.StatusOK, category)
}

func (api *API) transactions(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if _, ok := api.requireSession(w, r); !ok {
		return
	}
	transactions, err := api.service.ListTransactions(r.Context(), r.URL.Query().Get("month"))
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]any{"transactions": transactions})
}

func (api *API) createTransaction(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	member, ok := api.requireSession(w, r)
	if !ok {
		return
	}
	var input app.CreateTransactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	transaction, err := api.service.CreateTransaction(r.Context(), member, input)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusCreated, transaction)
}

func (api *API) updateTransaction(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	member, ok := api.requireSession(w, r)
	if !ok {
		return
	}
	var input app.UpdateTransactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	transaction, err := api.service.UpdateTransaction(r.Context(), member, chi.URLParam(r, "id"), input)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, transaction)
}

func (api *API) deleteTransaction(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	member, ok := api.requireSession(w, r)
	if !ok {
		return
	}
	if err := api.service.DeleteTransaction(r.Context(), member, chi.URLParam(r, "id")); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]bool{"ok": true})
}

func (api *API) monthlyOverview(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if _, ok := api.requireSession(w, r); !ok {
		return
	}
	overview, err := api.service.MonthlyOverview(r.Context(), r.URL.Query().Get("month"))
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, overview)
}

func (api *API) requireSession(w stdhttp.ResponseWriter, r *stdhttp.Request) (app.MemberDTO, bool) {
	session, err := api.service.SessionFromToken(r.Context(), sessionTokenFromRequest(r))
	if err != nil {
		writeJSON(w, stdhttp.StatusUnauthorized, map[string]string{"error": "login required"})
		return app.MemberDTO{}, false
	}
	return session.Member, true
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

func writeServiceError(w stdhttp.ResponseWriter, err error) {
	status := stdhttp.StatusBadRequest
	if strings.Contains(err.Error(), "permission") {
		status = stdhttp.StatusForbidden
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
