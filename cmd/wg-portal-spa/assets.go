package main

import "embed"

//go:embed frontend-dist/assets/*
//go:embed frontend-dist/img/*
//go:embed frontend-dist/index.html
//go:embed frontend-dist/favicon.ico
//go:embed frontend-dist/favicon.png
//go:embed frontend-dist/favicon-large.png
var frontendStatics embed.FS
