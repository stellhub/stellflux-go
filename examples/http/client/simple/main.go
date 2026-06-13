package main

import (
	"log"

	"github.com/stellhub/stellar"
	"github.com/stellhub/stellar/examples/http/client/simple/internal"
)

func main() {
	if _, err := stellar.Start(stellar.WithStarter(internal.NewClientStarter())); err != nil {
		log.Fatal(err)
	}
}
