package main

import (
	"log"

	"github.com/netlify/authlify/cmd"
)

func main() {
	if err := cmd.RootCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
