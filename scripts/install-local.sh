#!/bin/bash
# Quick script to rebuild and install sietch locally

set -e

echo "🔨 Building sietch..."
go build -o sietch ./main.go

echo "📦 Installing to ~/.local/bin..."
cp sietch /home/nilay/.local/bin/sietch

echo "🧹 Cleaning up..."
rm sietch

echo "✅ Sietch installed successfully!"
echo "📍 Location: /home/nilay/.local/bin/sietch"
echo "🔍 Test it out: sietch --help"
