package main

import (
	"log"

	"github.com/stellhub/stellar"
	"github.com/stellhub/stellar/examples/discovery/simple/internal"
)

func main() {
	if err := stellar.Run(stellar.WithStarter(internal.NewDiscoveryStarter())); err != nil {
		log.Fatal(err)
	}
}
