package utils

const (
	KMSEncryptionPublicKeyPath  = "keys/%s/kms-encryption-public-key.pem"
	KMSSigningPublicKeyPath     = "keys/%s/kms-signing-public-key.pem"
	KMSClientPath               = "tools/kms-client"
	LayeredDockerfilePath       = "internal/templates/docker/Dockerfile.layered.tmpl"
	EnvSourceScriptTemplatePath = "internal/templates/scripts/compute-source-env.sh.tmpl"
	JWTFilePath                 = "/run/container_launcher/attestation_verifier_claims_token"

	// Build-related constants
	TempImagePrefix       = "eigenx-temp-"
	LayeredBuildDirPrefix = "eigenx-layered-build"
	LayeredDockerfileName = "Dockerfile.eigencompute"
	EnvSourceScriptName   = "compute-source-env.sh"
	KMSClientBinaryName   = "kms-client"
	KMSEncryptionKeyName  = "kms-encryption-public-key.pem"
	KMSSigningKeyName     = "kms-signing-public-key.pem"
	TlsKeygenBinaryName   = "tls-keygen"
	CaddyfileName         = "Caddyfile"
	DockerPlatform        = "linux/amd64"
	LinuxOS               = "linux"
	AMD64Arch             = "amd64"
	SHA256Prefix          = "sha256:"

	RegistryPropagationWaitSeconds = 3
)

type LayeredDockerfileTemplateData struct {
	BaseImage        string
	OriginalCmd      string
	OriginalUser     string
	LogRedirect      string
	IncludeTLS       bool
	EigenXCLIVersion string
}

type EnvSourceScriptTemplateData struct {
	KMSServerURL string
	JWTFile      string
	UserAPIURL   string
}
