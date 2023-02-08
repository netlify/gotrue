package main

import (
	"log"

	"github.com/netlify/gotrue/cmd"
	tconf "github.com/tigrisdata/tigris-client-go/config"
	"github.com/tigrisdata/tigris-client-go/driver"
	"context"
)

func main() {
	drv, _ := driver.NewDriver(context.TODO(), &tconf.Driver{URL: cmd.TigrisConfig.URL})

	_, _ = drv.CreateProject(context.TODO(), "gotrue")

	if err := cmd.RootCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
