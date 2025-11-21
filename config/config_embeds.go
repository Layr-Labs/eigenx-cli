package config

// Note: The 'embed' package import is automatically handled by the Go compiler 
// when the 'go:embed' directive is used in Go 1.16+. 

// --- General Project Files ---

//go:embed .gitignore
var GitIgnore string // Contents of the project's .gitignore file.

//go:embed README.md
var ReadmeContent string // Contents of the project's main README file.

// --- Environment Examples and Configuration Templates ---

//go:embed .env.example
var EnvExample string // Example environment variables for general configuration.

//go:embed tls/.env.example.tls
var EnvExampleTLS string // Example environment variables specific to TLS/SSL setup.

//go:embed tls/Caddyfile.tmpl
var CaddyfileTLS string // Caddy web server configuration template for TLS setup.
