# EigenX CLI Installation Script for Windows

param(
    [switch]$Dev,
    [switch]$Interactive
)

# Error handling
$ErrorActionPreference = "Stop"

# Initialize cleanup variable
$tempFile = $null

# Check for --dev flag (support both -Dev switch and --dev argument)
if ($Dev -or ($args.Count -gt 0 -and $args[0] -eq "--dev")) {
    Write-Host "Installing dev version of EigenX..."
    # Dev version - read from dev S3 bucket
    $EIGENX_VERSION = (Invoke-WebRequest -Uri "https://s3.amazonaws.com/eigenlayer-eigenx-releases-dev/VERSION" -UseBasicParsing).Content.Trim()
    $EIGENX_BASE_URL = "https://s3.amazonaws.com/eigenlayer-eigenx-releases-dev"
} else {
    Write-Host "Installing stable version of EigenX..."
    # Stable version - read from prod S3 bucket
    $EIGENX_VERSION = (Invoke-WebRequest -Uri "https://s3.amazonaws.com/eigenlayer-eigenx-releases/VERSION" -UseBasicParsing).Content.Trim()
    $EIGENX_BASE_URL = "https://s3.amazonaws.com/eigenlayer-eigenx-releases"
}

# Detect platform
$OS = "windows"
$ARCH = $env:PROCESSOR_ARCHITECTURE

switch ($ARCH) {
    "AMD64" { $ARCH = "amd64" }
    "ARM64" { $ARCH = "arm64" }
    "x86" {
        Write-Host "Error: 32-bit Windows is not supported"
        exit 1
    }
    default {
        Write-Host "Error: Unsupported architecture: $ARCH"
        exit 1
    }
}

$PLATFORM = "${OS}-${ARCH}"

# Function to test if running as administrator
function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Prompt for installation directory
# Check if we're in a truly interactive session or explicitly requested
$isInteractive = $Interactive -or ([Environment]::UserInteractive -and -not [Console]::IsInputRedirected)

if ($isInteractive) {
    # Interactive session available
    Write-Host "Where would you like to install EigenX?"
    Write-Host "1) $env:USERPROFILE\bin (recommended)"
    Write-Host "2) C:\Program Files\EigenX (system-wide, requires admin)"
    Write-Host "3) Custom path"

    do {
        $choice = Read-Host "Enter choice (1-3) [1]"
        if ([string]::IsNullOrWhiteSpace($choice)) {
            $choice = "1"
        }
    } while ($choice -notin @("1", "2", "3"))
} else {
    # Non-interactive (piped), use default
    Write-Host "Installing to $env:USERPROFILE\bin (default for non-interactive install)"
    $choice = "1"
}

switch ($choice) {
    "1" {
        $INSTALL_DIR = "$env:USERPROFILE\bin"
        $needsAdmin = $false
    }
    "2" {
        $INSTALL_DIR = "C:\Program Files\EigenX"
        $needsAdmin = $true
    }
    "3" {
        do {
            $INSTALL_DIR = Read-Host "Enter custom path"
        } while ([string]::IsNullOrWhiteSpace($INSTALL_DIR))

        # Check if custom path requires admin (system directories)
        $needsAdmin = $INSTALL_DIR.StartsWith("C:\Program Files") -or
                     $INSTALL_DIR.StartsWith("C:\Windows") -or
                     $INSTALL_DIR.StartsWith("C:\ProgramData")
    }
}

# Check admin privileges if needed
if ($needsAdmin -and -not (Test-Administrator)) {
    Write-Host "Error: Installation to '$INSTALL_DIR' requires administrator privileges."
    Write-Host "Please run this script as Administrator or choose a different installation directory."
    exit 1
}

# Create directory if it doesn't exist
if (-not (Test-Path $INSTALL_DIR)) {
    try {
        New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
    } catch {
        Write-Host "Error: Failed to create directory '$INSTALL_DIR': $($_.Exception.Message)"
        exit 1
    }
}

# Download and install
$EIGENX_URL = "${EIGENX_BASE_URL}/${EIGENX_VERSION}/eigenx-cli-${PLATFORM}-${EIGENX_VERSION}.zip"
Write-Host "Downloading EigenX ${EIGENX_VERSION} for ${PLATFORM}..."

try {
    # Create temporary file for download
    $tempFile = [System.IO.Path]::GetTempFileName() + ".zip"

    # Download the binary file
    Invoke-WebRequest -Uri $EIGENX_URL -OutFile $tempFile -UseBasicParsing

    # Extract to installation directory
    Write-Host "Extracting to $INSTALL_DIR..."
    Expand-Archive -Path $tempFile -DestinationPath $INSTALL_DIR -Force

    Write-Host "✅ EigenX installed to $INSTALL_DIR\eigenx.exe"

} catch {
    Write-Host "Error: Failed to download or install EigenX: $($_.Exception.Message)"
    exit 1
} finally {
    # Clean up temp file
    if ($tempFile -and (Test-Path $tempFile)) {
        Remove-Item $tempFile -Force -ErrorAction SilentlyContinue
    }
}

# Add to PATH if needed
$currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($INSTALL_DIR -eq "$env:USERPROFILE\bin" -and $currentPath -notlike "*$INSTALL_DIR*") {
    Write-Host "💡 Adding $INSTALL_DIR to your PATH..."
    try {
        $newPath = "$INSTALL_DIR;$currentPath"
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        Write-Host "   PATH updated for current user. Restart your terminal to use 'eigenx' command."
    } catch {
        Write-Host "   Manual PATH setup required:"
        Write-Host "   Add '$INSTALL_DIR' to your PATH environment variable"
    }
} elseif ($INSTALL_DIR -eq "C:\Program Files\EigenX") {
    # For system-wide installation, add to system PATH
    try {
        $systemPath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
        if ($systemPath -notlike "*$INSTALL_DIR*") {
            Write-Host "💡 Adding $INSTALL_DIR to system PATH..."
            $newSystemPath = "$INSTALL_DIR;$systemPath"
            [Environment]::SetEnvironmentVariable("PATH", $newSystemPath, "Machine")
            Write-Host "   System PATH updated. Restart your terminal to use 'eigenx' command."
        }
    } catch {
        Write-Host "   Manual PATH setup required:"
        Write-Host "   Add '$INSTALL_DIR' to your system PATH environment variable"
    }
}

Write-Host "🚀 Verify installation: eigenx --help"
Write-Host "   (Restart your terminal if 'eigenx' command is not found)"
