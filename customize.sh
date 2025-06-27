#!/bin/bash

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <NEW_PROEJCT> <CLI_NAME>"
    echo "Example: $0 github.com/yourusername/yourproject yourcli"
    exit 1
fi

NEW_MODULE_PATH=$1
CLI_NAME=$2
ORIGINAL_MODULE_PATH="github.com/samzong/cli-template"
ORIGINAL_CLI_NAME="mycli"

echo "Starting project from $ORIGINAL_MODULE_PATH Modify to $NEW_MODULE_PATH..."
echo "CLI NAME from $ORIGINAL_CLI_NAME Modify to $CLI_NAME..."

echo "Update go.mod..."
sed -i '' "s|module $ORIGINAL_MODULE_PATH|module $NEW_MODULE_PATH|g" go.mod

echo "Update the import paths in all Go files..."
find . -type f -name "*.go" -exec sed -i '' "s|$ORIGINAL_MODULE_PATH|$NEW_MODULE_PATH|g" {} \;

echo "Update CLI Name..."
sed -i '' "s|CLI_NAME = \"$ORIGINAL_CLI_NAME\"|CLI_NAME = \"$CLI_NAME\"|g" cmd/root.go

echo "Update Makefile..."
sed -i '' "s|BINARY_NAME=$ORIGINAL_CLI_NAME|BINARY_NAME=$CLI_NAME|g" Makefile
sed -i '' "s|github.com/samzong/$ORIGINAL_CLI_NAME|github.com/samzong/$CLI_NAME|g" Makefile
sed -i '' "s|$ORIGINAL_MODULE_PATH|$NEW_MODULE_PATH|g" Makefile

echo "Update .goreleaser.yaml..."
sed -i '' "s|$ORIGINAL_MODULE_PATH|$NEW_MODULE_PATH|g" .goreleaser.yaml

echo "Update GitHub Actions Workflows..."
find .github/workflows -type f -name "*.yml" -exec sed -i '' "s|samzong/$ORIGINAL_CLI_NAME|samzong/$CLI_NAME|g" {} \;

echo "Update configuration file name reference..."
find . -type f -name "*.go" -exec sed -i '' "s|.$ORIGINAL_CLI_NAME.yaml|.$CLI_NAME.yaml|g" {} \;
find . -type f -name "*.go" -exec sed -i '' "s|.$ORIGINAL_CLI_NAME.yml|.$CLI_NAME.yml|g" {} \;
find . -type f -name "*.go" -exec sed -i '' "s|.$ORIGINAL_CLI_NAME.json|.$CLI_NAME.json|g" {} \;

echo "Update .gitignore..."
sed -i '' "s|.$ORIGINAL_CLI_NAME.yaml|.$CLI_NAME.yaml|g" .gitignore
sed -i '' "s|.$ORIGINAL_CLI_NAME.yml|.$CLI_NAME.yml|g" .gitignore
sed -i '' "s|.$ORIGINAL_CLI_NAME.json|.$CLI_NAME.json|g" .gitignore

echo "Sorting out and updating dependencies..."
go mod tidy

echo "✅ Project has been successfully updated!"
echo "  - Module path: $NEW_MODULE_PATH"
echo "  - CLI Name: $CLI_NAME"
