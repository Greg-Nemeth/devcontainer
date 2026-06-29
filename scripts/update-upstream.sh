#!/bin/bash
# scripts/update-upstream.sh
# This script fetches updates from the upstream devcontainers repositories
# and reports the latest tags/versions.

set -e

# Get workspace directory
WORKSPACE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "Checking upstream repositories..."

# Check spec
if [ -d "$WORKSPACE_DIR/spec" ]; then
    echo "Updating spec..."
    cd "$WORKSPACE_DIR/spec"
    git fetch --tags --all
    LATEST_SPEC=$(git tag -l | sort -V | tail -n 1)
    echo "Latest spec version: $LATEST_SPEC"
else
    echo "Warning: spec directory not found!"
fi

# Check cli
if [ -d "$WORKSPACE_DIR/cli" ]; then
    echo "Updating cli..."
    cd "$WORKSPACE_DIR/cli"
    git fetch --tags --all
    LATEST_CLI=$(git tag -l | sort -V | tail -n 1)
    echo "Latest CLI version: $LATEST_CLI"
else
    echo "Warning: cli directory not found!"
fi

echo "Done checking upstream updates."
