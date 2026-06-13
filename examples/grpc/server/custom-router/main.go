package main

import (
	"log"

	"github.com/stellhub/stellar"
	"github.com/stellhub/stellar/examples/grpc/server/custom-router/internal"
)

func main() {
	if _, err := stellar.Start(stellar.WithStarter(internal.NewCustomRouterStarter())); err != nil {
		log.Fatal(err)
	}
}
