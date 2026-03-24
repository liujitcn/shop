package assets

import "embed"

//go:embed openapi.yaml
var OpenApiData []byte

//go:embed all:web/*
var WebAssets embed.FS
