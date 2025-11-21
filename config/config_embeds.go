package config

// This package manages embedded static files, configurations, and templates 
// using the standard Go 'embed' directive.

import (
	_ "embed"
)

// =======================================================
// 1. PROJECT CONFIGURATION TEMPLATES (Development & Deployment)
// These files provide default settings and ignore rules for the repository structure.
// =======================================================

//go:embed .gitignore
var GitIgnore string

//go:embed .env.example
var EnvExample string


// =======================================================
// 2. DOCUMENTATION
// Core project documentation.
// =======================================================

//go:embed README.md
var ReadMe string


// =======================================================
// 3. TLS / Caddy TEMPLATES
// Configuration and environment examples specifically for Caddy web server and TLS setup.
// =======================================================

//go:embed tls/Caddyfile.tmpl
var CaddyfileTLS string

//go:embed tls/.env.example.tls
var EnvExampleTLS string
