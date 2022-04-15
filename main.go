package main

import (
	"context"
	"log"

	"github.com/spinup-host/spinup/internal/cmd"
)

var apiVersion = "dev"

func main() {
	ctx := context.Background()
	if err := cmd.Execute(ctx, apiVersion); err != nil {
		log.Fatal(err)
	}
}
