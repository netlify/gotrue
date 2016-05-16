package main

import (
	"log"

	"github.com/netlify/authlify/api"
	"github.com/netlify/authlify/db"
)

func main() {
	db, err := db.NewDB()
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	api := api.NewApi(db)

	log.Fatalf(api.ListenAndServe("0.0.0.0:9191"))
}
