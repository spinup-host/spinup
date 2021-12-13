package main

import (
	"context"
	"embed"
	"log"

	"github.com/spinup-host/internal/cmd"
)

var (
	apiVersion = "dev"
)

func main() {
	ctx := context.Background()
	if err := cmd.Execute(ctx, apiVersion); err != nil {
		log.Fatal(err)
	}
}

//go:embed templates/*
var DockerTempl embed.FS
