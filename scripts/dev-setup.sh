#!/bin/bash

# AI Chat Platform - Local Development Setup
# This script sets up the frontend for local development with real backend resources

set -e

echo "🚀 Setting up AI Chat Platform for local development..."

# Navigate to frontend directory
cd "$(dirname "$0")/../frontend"

echo "📦 Installing frontend dependencies..."
npm install

echo "⚙️  Configuring for local development..."
# Copy development config
cp next.config.dev.js next.config.js

echo "🔧 Setting up environment variables..."
# Create .env.local file with backend endpoints
cat > .env.local << EOF
# Local development environment variables
NODE_ENV=development
NEXT_PUBLIC_API_ENDPOINT=https://c9wu2knteb.execute-api.us-east-1.amazonaws.com/prod/
NEXT_PUBLIC_WS_ENDPOINT=wss://bh9bgcvljl.execute-api.us-east-1.amazonaws.com/prod
NEXT_PUBLIC_USER_POOL_ID=us-east-1_6QZ8ANScT
NEXT_PUBLIC_USER_POOL_CLIENT_ID=5gf3gocpg5eu3c01ipeee4tib2
NEXT_PUBLIC_REGION=us-east-1
EOF

echo "✅ Local development setup complete!"
echo ""
echo "🎉 You can now start the development server with:"
echo "   cd frontend && npm run dev"
echo ""
echo "🌐 The app will be available at: http://localhost:3000"
echo "🔗 It will connect to your real backend at: https://c9wu2knteb.execute-api.us-east-1.amazonaws.com/prod/"
echo ""
echo "📝 For production deployment, run:"
echo "   ./scripts/update-frontend.sh"