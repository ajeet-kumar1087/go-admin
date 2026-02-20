package main

import (
	"fmt"
	"github.com/ajeet-kumar1087/go-admin/pkg/admin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"html/template"
	"log"
	"net/http"
)

type Role struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"uniqueIndex"`
}

type User struct {
	ID    uint   `gorm:"primaryKey"`
	Email string
	Role  string
}

type Product struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string
	Price float64
	Image string
}

type ProductInfo struct {
	ID          uint `gorm:"primaryKey"`
	ProductID   uint
	Description string
	Manufacturer string
}

func main() {
	db, err := gorm.Open(sqlite.Open("admin.db"), &gorm.Config{})
	if err != nil { log.Fatal("failed to connect database") }

	db.AutoMigrate(&User{}, &Product{}, &ProductInfo{}, &admin.Permission{}, &Role{}, &admin.AdminUser{}, &admin.Session{}, &admin.AuditLog{})

	adm := admin.NewRegistry(db)
	conf, _ := admin.LoadConfig("admin.yml")
	if conf != nil { adm.SetConfig(conf) }

	roles := []string{"admin", "editor", "viewer"}

	addActivityAction := func(r *admin.Resource) {
		r.AddMemberAction("activity", "View History", func(res *admin.Resource, w http.ResponseWriter, r *http.Request) {
			id := r.URL.Query().Get("id")
			http.Redirect(w, r, fmt.Sprintf("/admin/AuditLog?q_ResourceName=%s&q_RecordID=%s", res.Name, id), 303)
		})
	}

	// Administration Group
	adm.Register(admin.AdminUser{}).SetGroup("Administration").RegisterField("ID", "ID", true).RegisterField("Email", "Email", false).RegisterField("Role", "Role", false).SetFieldType("Role", "select", roles...)
	adm.Register(admin.AuditLog{}).SetGroup("Administration").RegisterField("ID", "ID", true).RegisterField("CreatedAt", "Time", true).RegisterField("UserEmail", "User", true).RegisterField("ResourceName", "Resource", true).RegisterField("RecordID", "Record ID", true).RegisterField("Action", "Action", true).RegisterField("Changes", "Changes", true)
	adm.Register(Role{}).SetGroup("Administration").RegisterField("ID", "ID", true).RegisterField("Name", "Role Name", false)
	adm.Register(admin.Permission{}).SetGroup("Administration").RegisterField("ID", "ID", true).RegisterField("Role", "Role Name", false).RegisterField("ResourceName", "Resource", false).RegisterField("Action", "Action", false).SetFieldType("Role", "select", roles...).SetFieldType("ResourceName", "select", adm.ResourceNames()...).SetFieldType("Action", "select", "list", "show", "new", "edit", "save", "delete")

	// Users
	uRes := adm.Register(User{}).
		RegisterField("ID", "ID", true).
		RegisterField("Email", "Email Address", false).
		RegisterField("Role", "User Role", false).
		SetFieldType("Role", "select", roles...).
		SetDecorator("Role", func(val interface{}) template.HTML {
			role := val.(string); color := "#64748b"
			if role == "admin" { color = "#ef4444" } else if role == "editor" { color = "#3b82f6" }
			return template.HTML(fmt.Sprintf(`<span style="background: %s; color: white; padding: 0.2rem 0.5rem; border-radius: 9999px; font-size: 0.75rem; font-weight: 600;">%s</span>`, color, role))
		}).
		AddSidebar("Quick Actions", func(res *admin.Resource, item interface{}) template.HTML {
			u := item.(*User)
			return template.HTML(fmt.Sprintf(`<div style="display: flex; flex-direction: column; gap: 0.5rem;"><a href="mailto:%s" class="btn" style="text-align: center; background: #f1f5f9;">Email User</a></div>`, u.Email))
		})
	addActivityAction(uRes)

	// Products
	pRes := adm.Register(Product{}).
		SetGroup("Products").
		RegisterField("ID", "ID", true).
		RegisterField("Name", "Product Name", false).
		RegisterField("Price", "Price", false).
		RegisterField("Image", "Product Image", false).
		SetFieldType("Price", "number").
		SetFieldType("Image", "image").
		SetSortable("Image", false).
		SetDecorator("Price", func(val interface{}) template.HTML {
			return template.HTML(fmt.Sprintf("<strong>$%.2f</strong>", val.(float64)))
		}).
		AddSidebar("Market Info", func(res *admin.Resource, item interface{}) template.HTML {
			return template.HTML(`<div style="font-size: 0.8125rem; color: #475569;"><p>Competitor Avg: $145.00</p><p style="color: #10b981; margin-top: 0.25rem;">+12%% vs last month</p></div>`)
		}).
		HasMany("ProductInfo", "Technical Specifications", "ProductInfo", "ProductID").
		AddCollectionAction("discount", "Apply 10% Bulk Discount", func(res *admin.Resource, w http.ResponseWriter, r *http.Request) {
			db.Model(&Product{}).Where("price > ?", 0).Update("price", gorm.Expr("price * 0.9"))
			http.Redirect(w, r, "/admin/Product", 303)
		}).
		AddBatchAction("batch_delete", "Delete Selected", func(res *admin.Resource, ids []string, w http.ResponseWriter, r *http.Request) {
			db.Where("id IN ?", ids).Delete(&Product{})
			http.Redirect(w, r, "/admin/Product", 303)
		})
	addActivityAction(pRes)

	adm.Register(ProductInfo{}).SetGroup("Products").RegisterField("ID", "ID", true).RegisterField("ProductID", "Product", false).RegisterField("Description", "Description", false).RegisterField("Manufacturer", "Manufacturer", false).BelongsTo("ProductID", "Parent Product", "Product", "ID").SetSearchable("ProductID", "Product")

	// Charts
	adm.AddChart("Users by Role", "pie", func(db *gorm.DB) ([]string, []float64) {
		var results []struct { Role string; Count int64 }; db.Model(&User{}).Select("role, count(*) as count").Group("role").Scan(&results)
		labels := []string{}; values := []float64{}; for _, r := range results { labels, values = append(labels, r.Role), append(values, float64(r.Count)) }; return labels, values
	})

	// Custom Pages
	adm.AddPage("SystemStatus", "Administration", func(w http.ResponseWriter, r *http.Request) {
		content := template.HTML(`<div style="background: white; border-radius: 0.5rem; overflow: hidden;"><table style="width: 100%; border-collapse: collapse;"><tr style="border-bottom: 1px solid #e2e8f0;"><td style="padding: 1rem; font-weight: 600;">Server Status</td><td style="padding: 1rem; color: #10b981;">Online</td></tr></table></div>`)
		adm.RenderCustomPage(w, r, "System Status", content)
	})

	// Seed Data
	var adminCount int64; db.Model(&admin.AdminUser{}).Count(&adminCount)
	if adminCount == 0 {
		adminUser := &admin.AdminUser{Email: "admin@example.com", Role: "admin"}
		adminUser.SetPassword("password123"); db.Create(adminUser)
		db.Create(&Role{Name: "admin"}); db.Create(&Role{Name: "editor"}); db.Create(&Role{Name: "viewer"})
		db.Create(&admin.Permission{Role: "editor", ResourceName: "Product", Action: "list"})
		p1 := &Product{Name: "Mechanical Keyboard", Price: 150.00}; db.Create(p1)
		db.Create(&ProductInfo{ProductID: p1.ID, Description: "Blue Switches", Manufacturer: "Razer"})
		db.Create(&User{Email: "user@example.com", Role: "editor"})
	}

	fmt.Printf("\n🚀 Admin running at http://localhost:8080/admin\n")
	http.Handle("/admin/", adm)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
