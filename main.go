package main

import (
	"fmt"
	"log"

	"github.com/netlify/authlify/api"
	"github.com/netlify/authlify/conf"
	"github.com/netlify/authlify/mailer"
	"github.com/netlify/authlify/models"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // Import SQLite
)

func main() {
	config, err := conf.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := gorm.Open("sqlite3", "gorm.db")
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	db.AutoMigrate(&models.User{}, &models.RefreshToken{})

	mailer := mailer.NewMailer(config)

	api := api.NewAPI(config, db, mailer)

	api.ListenAndServe(fmt.Sprintf("%v:%v", config.API.Host, config.API.Port))
}
