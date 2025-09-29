//go:build !prod
// +build !prod

package project

import (
	"embed"
)

//go:embed keys/*/dev/*
var KeysFS embed.FS
