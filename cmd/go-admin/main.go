package main

import (
	"fmt"
	"os"
	"strings"
)

const helpText = `Go Admin CLI - Scaffolding Tool

Usage:
  go-admin init              Scaffold a new admin project
  go-admin generate <name>   Generate boilerplate for a resource
`

const mainTemplate = `package main

import (
	"github.com/ajeet-kumar1087/go-admin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"net/http"
)

func main() {
	db, _ := gorm.Open(sqlite.Open("admin.db"), &gorm.Config{})
	
	adm := admin.NewRegistry(db)
	
	// Load Custom Config
	conf, _ := admin.LoadConfig("admin.yml")
	adm.SetConfig(conf)

	log.Println("🚀 Admin panel starting on http://localhost:8080/admin")
	http.Handle("/admin/", adm)
	http.ListenAndServe(":8080", nil)
}
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(helpText)
		return
	}

	cmd := os.Args[1]

	switch cmd {
	case "init":
		handleInit()
	case "generate":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing resource name. Usage: go-admin generate <ResourceName>")
			return
		}
		handleGenerate(os.Args[2])
	default:
		fmt.Print(helpText)
	}
}

func handleInit() {
	fmt.Println("Creating main.go...")
	os.WriteFile("main.go", []byte(mainTemplate), 0644)
	
	fmt.Println("Creating admin.yml...")
	os.WriteFile("admin.yml", []byte("site_title: \"My Admin\"\ndefault_per_page: 10\n"), 0644)
	
	fmt.Println("✅ Done! Run 'go mod init' and 'go mod tidy' to start.")
}

func handleGenerate(name string) {
	name = strings.Title(name)
	tmpl := fmt.Sprintf(`
// Registration for %s
adm.Register(%s{}).
	RegisterField("ID", "ID", true).
	RegisterField("Name", "Name", false)
`, name, name)

	fmt.Println("Boilerplate generated:")
	fmt.Println("-------------------")
	fmt.Println(tmpl)
	fmt.Println("-------------------")
	fmt.Println("Copy the above code into your main.go registration block.")
}
