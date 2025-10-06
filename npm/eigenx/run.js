#!/usr/bin/env node

const { spawn } = require('child_process');
const { join } = require('path');
const { existsSync } = require('fs');

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

function getBinaryPath() {
  const platform = process.platform;
  const arch = process.arch;
  const key = `${platform}-${arch}`;

  const packageName = PLATFORMS[key];
  const binaryName = BINARY_NAMES[key];

  if (!packageName || !binaryName) {
    console.error(`Unsupported platform: ${key}`);
    process.exit(1);
  }

  const binaryPath = join(__dirname, 'node_modules', packageName, 'bin', binaryName);

  if (!existsSync(binaryPath)) {
    console.error(`eigenx binary not found at: ${binaryPath}`);
    console.error('Please try reinstalling: npm install -g @layr-labs/eigenx');
    process.exit(1);
  }

  return binaryPath;
}

function run() {
  const binaryPath = getBinaryPath();
  const args = process.argv.slice(2);

  const child = spawn(binaryPath, args, {
    stdio: 'inherit',
    windowsHide: false
  });

  child.on('exit', (code, signal) => {
    if (signal) {
      process.kill(process.pid, signal);
    } else {
      process.exit(code);
    }
  });

  // Forward signals to child process
  process.on('SIGINT', () => {
    child.kill('SIGINT');
  });

  process.on('SIGTERM', () => {
    child.kill('SIGTERM');
  });
}

run();
