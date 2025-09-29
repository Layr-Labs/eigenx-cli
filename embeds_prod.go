//go:build prod
// +build prod

package project

import (
	"embed"
)

//go:embed keys/*/prod/*
var KeysFS embed.FS
