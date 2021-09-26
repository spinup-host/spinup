package templates

import (
	"embed"
)

//go:embed templates/*
var DockerTempl embed.FS
