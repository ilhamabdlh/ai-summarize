#!/bin/bash

echo "🚀 AI CV Summarize - AWS Deployment Script"
echo "==========================================="

# Update system
echo "📦 Updating system packages..."
sudo apt update && sudo apt upgrade -y

# Install Golang
echo "🔧 Installing Golang 1.21..."
cd ~
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# Add to PATH
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:~/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export PATH=$PATH:~/go/bin' >> ~/.bashrc

# Verify Go installation
go version

# Cleanup
rm go1.21.0.linux-amd64.tar.gz

# Install MongoDB
echo "🗄️ Installing MongoDB..."
curl -fsSL https://www.mongodb.org/static/pgp/server-7.0.asc | sudo gpg -o /usr/share/keyrings/mongodb-server-7.0.gpg --dearmor
echo "deb [ arch=amd64,arm64 signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/7.0 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-7.0.list
sudo apt update
sudo apt install -y mongodb-org

# Start MongoDB
sudo systemctl start mongod
sudo systemctl enable mongod

# Install Redis
echo "🔴 Installing Redis..."
sudo apt install -y redis-server
sudo systemctl start redis-server
sudo systemctl enable redis-server

# Install other dependencies
echo "📚 Installing other dependencies..."
sudo apt install -y git curl wget htop

# Build application
echo "🏗️ Building application..."
cd ~/api/ai-summarize
go mod download
go mod tidy
go build -o ai-cv-summarize ./cmd/server/main.go

# Create uploads directory
mkdir -p uploads

# Create log directory
sudo mkdir -p /var/log/ai-cv-summarize
sudo chown $USER:$USER /var/log/ai-cv-summarize

echo ""
echo "✅ Installation complete!"
echo ""
echo "📋 Next steps:"
echo "1. Configure .env file with your OpenAI API key"
echo "   nano .env"
echo ""
echo "2. Create systemd service:"
echo "   sudo nano /etc/systemd/system/ai-cv-summarize.service"
echo ""
echo "3. Start the service:"
echo "   sudo systemctl start ai-cv-summarize"
echo ""
echo "4. Check status:"
echo "   sudo systemctl status ai-cv-summarize"
echo ""
echo "🚀 Application ready to run!"
