# Go Admin

A high-performance, modern, and extensible administration framework for Go, inspired by Active Admin. 

Go Admin uses reflection and GORM to automatically generate a professional-grade back-office for your database models with minimal configuration.

## Features

- 🔐 **Secure Authentication**: Session-based login with bcrypt password hashing.
- 📂 **Resource Grouping**: Organize your models into logical categories.
- 📊 **Visual Dashboard**: Customizable charts (powered by Chart.js) and stat widgets.
- 🔍 **Powerful Filtering**: Predefined scopes (tabs) and dynamic search filters.
- ⛓️ **Associations**: Automatic handling of `HasMany` and `BelongsTo` relationships.
- 📝 **Audit Logging**: Full history of every Create, Update, and Delete action.
- 📦 **Batch Actions**: Perform operations on multiple records at once.
- 📥 **CSV Export**: Export filtered data directly to CSV.
- 🎨 **Decorators**: Customize how fields are rendered (Currency, Badges, etc.).
- 🚀 **Portable**: Everything (HTML/CSS/JS) is bundled into your binary using `go:embed`.

## Installation

```bash
go get github.com/ajeet-kumar1087/go-admin
```

## Quick Start

```go
package main

import (
    "github.com/ajeet-kumar1087/go-admin/pkg/admin"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "net/http"
)

type Product struct {
    ID    uint   `gorm:"primaryKey"`
    Name  string
    Price float64
}

func main() {
    db, _ := gorm.Open(sqlite.Open("admin.db"), &gorm.Config{})
    db.AutoMigrate(&Product{}, &admin.AdminUser{}, &admin.Permission{}, &admin.Session{}, &admin.AuditLog{})

    // Initialize Admin
    adm := admin.NewRegistry(db)

    // Register a Resource
    adm.Register(Product{}).
        SetGroup("Inventory").
        RegisterField("ID", "ID", true).
        RegisterField("Name", "Product Name", false).
        RegisterField("Price", "Price", false)

    // Start Server
    http.Handle("/admin/", adm)
    http.ListenAndServe(":8080", nil)
}
```

## Documentation

For full feature documentation including **Associations**, **Scopes**, **Custom Actions**, and **Charts**, please refer to the [Usage Guide](USAGE.md).

## License

This project is licensed under the MIT License.
