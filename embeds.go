package project

import (
	"embed"
)

//go:embed README.md
var RawReadme []byte

//go:embed tools/kms-client-linux-amd64
var RawKmsClient []byte

//go:embed tools/tls-keygen-linux-amd64
var RawTlsKeygenBinary []byte

//go:embed internal/templates/*
var TemplatesFS embed.FS
