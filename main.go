package main

import (
	"context"
	"log"

	"github.com/spinup-host/spinup/build"
	"github.com/spinup-host/spinup/internal/cmd"
)

func main() {
	ctx := context.Background()
	bi := build.Info{
		Version: build.Version,
		Commit:  build.FullCommit,
		Branch:  build.Branch,
	}
	if err := cmd.Execute(ctx, bi); err != nil {
		log.Fatal(err)
	}
}
