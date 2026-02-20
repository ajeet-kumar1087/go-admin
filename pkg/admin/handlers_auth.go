package admin

import (
	"github.com/google/uuid"
	"github.com/ajeet-kumar1087/go-admin/pkg/admin/models"
	"html/template"
	"net/http"
	"time"
)

func (reg *Registry) IsAllowed(role, resource, action string) bool {
	if role == "admin" { return true }
	var count int64
	reg.DB.Model(&models.Permission{}).Where("role = ? AND resource_name = ? AND action = ?", role, resource, action).Count(&count)
	return count > 0
}

func (reg *Registry) GetUserFromRequest(r *http.Request) (*models.AdminUser, string) {
	cookie, err := r.Cookie("admin_session")
	if err != nil { return nil, "guest" }
	var sess models.Session
	if err := reg.DB.Where("id = ? AND expires_at > ?", cookie.Value, time.Now()).First(&sess).Error; err != nil { return nil, "guest" }
	var user models.AdminUser
	if err := reg.DB.First(&user, sess.UserID).Error; err != nil { return nil, "guest" }
	return &user, user.Role
}

func (reg *Registry) handleLogin(w http.ResponseWriter, r *http.Request) {
	email, password := r.FormValue("email"), r.FormValue("password")
	var user models.AdminUser
	if err := reg.DB.Where("email = ?", email).First(&user).Error; err != nil { reg.renderLogin(w, r, "Invalid credentials"); return }
	if !user.CheckPassword(password) { reg.renderLogin(w, r, "Invalid credentials"); return }
	sessionID := uuid.New().String()
	reg.DB.Create(&models.Session{ID: sessionID, UserID: user.ID, ExpiresAt: time.Now().Add(time.Duration(reg.Config.SessionTTL) * time.Hour)})
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: sessionID, Path: "/admin", HttpOnly: true})
	reg.setFlash(w, "Login successful! Welcome back.")
	http.Redirect(w, r, "/admin", 303)
}

func (reg *Registry) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("admin_session")
	if cookie != nil { reg.DB.Delete(&models.Session{}, "id = ?", cookie.Value) }
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "", Path: "/admin", Expires: time.Unix(0, 0), HttpOnly: true})
	http.Redirect(w, r, "/admin/login", 303)
}

func (reg *Registry) renderLogin(w http.ResponseWriter, r *http.Request, errorMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.ParseFS(templateFS, "templates/login.html"))
	styleContent, _ := templateFS.ReadFile("templates/style.css")
	tmpl.Execute(w, PageData{SiteTitle: reg.Config.SiteTitle, Error: errorMsg, CSS: template.CSS(styleContent)})
}
