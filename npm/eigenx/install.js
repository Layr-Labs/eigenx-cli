#!/usr/bin/env node

const { existsSync } = require('fs');
const { join } = require('path');

// Map Node.js platform/arch to package names
const PLATFORMS = {
  'darwin-arm64': '@layr-labs/eigenx-darwin-arm64',
  'darwin-x64': '@layr-labs/eigenx-darwin-amd64',
  'linux-arm64': '@layr-labs/eigenx-linux-arm64',
  'linux-x64': '@layr-labs/eigenx-linux-amd64',
  'win32-x64': '@layr-labs/eigenx-windows-amd64',
  'win32-arm64': '@layr-labs/eigenx-windows-arm64'
};

// Map platform to binary extension
const BINARY_NAMES = {
  'darwin-arm64': 'eigenx',
  'darwin-x64': 'eigenx',
  'linux-arm64': 'eigenx',
  'linux-x64': 'eigenx',
  'win32-x64': 'eigenx.exe',
  'win32-arm64': 'eigenx.exe'
};

function getPlatformPackage() {
  const platform = process.platform;
  const arch = process.arch;
  const key = `${platform}-${arch}`;

  return {
    packageName: PLATFORMS[key],
    binaryName: BINARY_NAMES[key],
    platformKey: key
  };
}

function validateInstallation() {
  const { packageName, binaryName, platformKey } = getPlatformPackage();

  if (!packageName) {
    console.error(`Unsupported platform: ${platformKey}`);
    console.error('eigenx is only supported on:');
    console.error('  - macOS (darwin): arm64, x64');
    console.error('  - Linux: arm64, x64');
    console.error('  - Windows: x64, arm64');
    process.exit(1);
  }

  // Check if the platform-specific binary exists
  const binaryPath = join(__dirname, 'node_modules', packageName, 'bin', binaryName);

  if (!existsSync(binaryPath)) {
    console.error(`Failed to install @layr-labs/eigenx for ${platformKey}`);
    console.error(`Expected binary at: ${binaryPath}`);
    console.error('');
    console.error('This may happen if:');
    console.error('  1. The platform-specific package failed to install');
    console.error('  2. Your platform is not supported');
    console.error('');
    console.error('Please report this issue at: https://github.com/Layr-Labs/eigenx-cli/issues');
    process.exit(1);
  }

  console.log(`✓ @layr-labs/eigenx installed successfully for ${platformKey}`);
}

validateInstallation();
