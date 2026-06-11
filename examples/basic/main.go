package main

import (
	"context"
	"log"

	"github.com/stellhub/stellar"
)

func main() {
	app := stellar.New(stellar.Config{
		AppName:     "basic-example",
		Environment: stellar.EnvDev,
		Zone:        "local",
	})
	app.Use(stellar.StandardModules()...)

	if err := app.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
	defer app.Stop(context.Background())
}
