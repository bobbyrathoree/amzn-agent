#!/bin/bash

# Cleanup Old Frontend Files
echo "🧹 Cleaning up old NextJS frontend files..."

# Remove NextJS frontend directory if it exists
if [ -d "/Users/bobbyrathore/Documents/WildProjects/experimental/frontend" ]; then
    echo "Removing old NextJS frontend directory..."
    rm -rf "/Users/bobbyrathore/Documents/WildProjects/experimental/frontend"
    echo "✅ Removed frontend directory"
fi

# Remove old frontend stack
if [ -f "/Users/bobbyrathore/Documents/WildProjects/experimental/infrastructure/lib/frontend-stack.ts" ]; then
    echo "Backing up old frontend stack..."
    mv "/Users/bobbyrathore/Documents/WildProjects/experimental/infrastructure/lib/frontend-stack.ts" \
       "/Users/bobbyrathore/Documents/WildProjects/experimental/infrastructure/lib/frontend-stack-old.ts.bak"
    echo "✅ Backed up old frontend stack"
fi

# Remove simple frontend stack
if [ -f "/Users/bobbyrathore/Documents/WildProjects/experimental/infrastructure/lib/simple-frontend-stack.ts" ]; then
    echo "Removing simple frontend stack..."
    rm "/Users/bobbyrathore/Documents/WildProjects/experimental/infrastructure/lib/simple-frontend-stack.ts"
    echo "✅ Removed simple frontend stack"
fi

# Rename new frontend stack
if [ -f "/Users/bobbyrathore/Documents/WildProjects/experimental/infrastructure/lib/frontend-stack-new.ts" ]; then
    echo "Activating new frontend stack..."
    mv "/Users/bobbyrathore/Documents/WildProjects/experimental/infrastructure/lib/frontend-stack-new.ts" \
       "/Users/bobbyrathore/Documents/WildProjects/experimental/infrastructure/lib/frontend-stack.ts"
    echo "✅ Activated new frontend stack"
fi

# Clean up CDK output
if [ -d "/Users/bobbyrathore/Documents/WildProjects/experimental/infrastructure/cdk.out" ]; then
    echo "Cleaning CDK output..."
    rm -rf "/Users/bobbyrathore/Documents/WildProjects/experimental/infrastructure/cdk.out"
    echo "✅ Cleaned CDK output"
fi

echo "🎉 Cleanup complete!"