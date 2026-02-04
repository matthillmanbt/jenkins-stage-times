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
echo "1. Setup bt cli:"
echo "   **Add ``jenkins-api`` to your personal vault**"
echo ""
echo "   mkdir -p ~/.sra-con/env && cat <<EOF > ~/.sra-con/env/jenkins.env"
echo "   JENKINS_HOST=bt-vault://personal/jenkins-api/description"
echo "   JENKINS_USER=bt-vault://personal/jenkins-api/username"
echo "   JENKINS_KEY=bt-vault://personal/jenkins-api/password"
echo "   EOF"
echo ""
echo "   alias jenkins=\"\$HOME/.sra-con/CLI/bt vault run \$HOME/.sra-con/env/jenkins.env -- jenkins\""
echo ""
echo "2. OR set environment variables directly:"
echo "   export JENKINS_HOST=https://your-jenkins.com"
echo "   export JENKINS_USER=your-username"
echo "   export JENKINS_KEY=your-api-key"
echo ""
echo "3. Or create ~/.jenkins.yaml with your configuration"
echo ""
echo "4. Test the installation:"
echo "   jenkins --version"
echo "   jenkins --help"
echo "   jenkins latest --pra"
echo ""
echo "5. Use in Claude Code:"
echo "   Ask Claude to 'start a build' or 'diagnose build 1234'"
