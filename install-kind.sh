#!/bin/bash

# Detect the operating system
OS=$(uname | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Verify whether kind has been installed or not
if command -v kind &> /dev/null; then
  echo "kind already installed"
  exit 0
else echo "insalling kind"
fi

# Decide which to download kind from
case "$OS" in
  linux*)
    case "$ARCH" in 
      x86_64*)
        KIND_URL="https://kind.sigs.k8s.io/dl/v0.22.0/kind-linux-amd64"
        ;;
      aarch64*)
        KIND_URL="https://kind.sigs.k8s.io/dl/v0.22.0/kind-linux-arm64"
        ;;
      *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
    esac
    DEST_FOLDER="/usr/local/bin"
    ;;
  darwin*)
    case "$ARCH" in 
      x86_64*)
        KIND_URL="https://kind.sigs.k8s.io/dl/v0.22.0/kind-darwin-amd64"
        ;;
      arm64*)
        KIND_URL="https://kind.sigs.k8s.io/dl/v0.22.0/kind-darwin-arm64"
        ;;
      *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
    esac
    DEST_FOLDER="/usr/local/bin"
    ;;
  *)
    echo "Unsupported operating system: $OS"
    exit 1
    ;;
esac

# Download the Kind binary
curl -Lo ./kind "$KIND_URL"
chmod +x ./kind

# Move the Kind binary to /usr/local/bin
mv ./kind "$DEST_FOLDER"/kind

# Verify the installation
kind version
