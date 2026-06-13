package main

import (
	"log"

	"github.com/stellhub/stellar"
)

func main() {
	if err := stellar.Run(); err != nil {
		log.Fatal(err)
	}
}
