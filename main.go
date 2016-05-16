package main

import (
	"fmt"
	"log"

	"github.com/netlify/authlify/api"
	"github.com/netlify/authlify/conf"
	"github.com/netlify/authlify/mailer"
	"github.com/netlify/authlify/models"

	_ "github.com/jinzhu/gorm/dialects/sqlite" // Import SQLite
)

func main() {
	config, err := conf.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := models.Connect(config)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	mailer := mailer.NewMailer(config)

	api := api.NewAPI(config, db, mailer)

	api.ListenAndServe(fmt.Sprintf("%v:%v", config.API.Host, config.API.Port))
}
