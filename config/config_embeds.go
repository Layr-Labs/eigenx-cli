package config

import _ "embed"

//go:embed .gitignore
var GitIgnore string

//go:embed .env.example
var EnvExample string

//go:embed README.md
var ReadMe string
//go:embed tls/Caddyfile.tmpl
var CaddyfileTLS string

//go:embed tls/.env.example.tls
var EnvExampleTLS string
