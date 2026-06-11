package main

import (
	"log"

	"github.com/stellhub/stellar"
)

func main() {
	if err := stellar.Start(); err != nil {
		log.Fatal(err)
	}
}
