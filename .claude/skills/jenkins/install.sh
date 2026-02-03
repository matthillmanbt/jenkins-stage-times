#!/bin/bash
# Jenkins Skill Installation Script

set -e

echo "Installing Jenkins CLI skill..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is required but not installed"
    echo "Install Go from https://golang.org/dl/"
    exit 1
fi

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

echo "Building jenkins binary..."
cd "$REPO_ROOT"
go build -o jenkins .

# Determine install location
if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

echo "Installing to $INSTALL_DIR..."
cp jenkins "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/jenkins"

# Check if install dir is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "⚠️  Warning: $INSTALL_DIR is not in your PATH"
    echo "Add this to your ~/.bashrc or ~/.zshrc:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo ""
echo "✓ Jenkins CLI installed successfully!"
echo ""
echo "Next steps:"
echo "1. Set environment variables:"
echo "   export JENKINS_HOST=https://your-jenkins.com"
echo "   export JENKINS_USER=your-username"
echo "   export JENKINS_KEY=your-api-key"
echo ""
echo "2. Or create ~/.jenkins.yaml with your configuration"
echo ""
echo "3. Test the installation:"
echo "   jenkins --version"
echo "   jenkins --help"
echo ""
echo "4. Use in Claude Code:"
echo "   Ask Claude to 'start a build' or 'diagnose build 1234'"
