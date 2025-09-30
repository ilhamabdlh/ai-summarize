# 🚀 AWS Deployment Guide - AI CV Summarize

## 📋 Prerequisites

- AWS EC2 instance (Ubuntu 22.04 LTS recommended)
- SSH access to your EC2 instance
- Domain/IP for accessing the API (optional)

## 🔧 Step 1: Install Golang

```bash
# Update system packages
sudo apt update && sudo apt upgrade -y

# Download and install Go 1.21
cd ~
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz

# Remove old Go installation if exists
sudo rm -rf /usr/local/go

# Extract and install
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# Add Go to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export PATH=$PATH:~/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
# Should output: go version go1.21.0 linux/amd64

# Cleanup
rm go1.21.0.linux-amd64.tar.gz
```

## 🗄️ Step 2: Install MongoDB

```bash
# Import MongoDB GPG key
curl -fsSL https://www.mongodb.org/static/pgp/server-7.0.asc | \
   sudo gpg -o /usr/share/keyrings/mongodb-server-7.0.gpg --dearmor

# Add MongoDB repository
echo "deb [ arch=amd64,arm64 signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/7.0 multiverse" | \
   sudo tee /etc/apt/sources.list.d/mongodb-org-7.0.list

# Update package list
sudo apt update

# Install MongoDB
sudo apt install -y mongodb-org

# Start MongoDB service
sudo systemctl start mongod
sudo systemctl enable mongod

# Verify MongoDB is running
sudo systemctl status mongod

# Test connection
mongosh --eval "db.version()"
```

## 🔴 Step 3: Install Redis

```bash
# Install Redis
sudo apt install -y redis-server

# Configure Redis to start on boot
sudo systemctl enable redis-server

# Start Redis
sudo systemctl start redis-server

# Verify Redis is running
sudo systemctl status redis-server

# Test Redis
redis-cli ping
# Should output: PONG
```

## 🐳 Step 4: Install Docker (Optional)

```bash
# Install Docker
sudo apt install -y apt-transport-https ca-certificates curl software-properties-common
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io

# Add user to docker group
sudo usermod -aG docker $USER

# Start Docker
sudo systemctl start docker
sudo systemctl enable docker

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Verify
docker --version
docker-compose --version
```

## 📦 Step 5: Clone and Setup Project

```bash
# Navigate to project directory
cd ~/api/ai-summarize

# Install dependencies
go mod download
go mod tidy

# Create .env file
cp env.example .env
nano .env
```

**Edit .env file:**
```env
# OpenAI Configuration
OPENAI_API_KEY=your-actual-openai-api-key-here
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=gpt-4

# Server Configuration
PORT=8080
GIN_MODE=release

# MongoDB Configuration
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=ai_cv_summarize

# Redis Configuration
REDIS_URL=redis://localhost:6379

# File Upload Configuration
MAX_FILE_SIZE=10485760
UPLOAD_DIR=./uploads

# Job Queue Configuration
JOB_TIMEOUT=300
MAX_RETRIES=3
```

## 🏗️ Step 6: Build Application

```bash
cd ~/api/ai-summarize

# Build the application
go build -o ai-cv-summarize ./cmd/server/main.go

# Make it executable
chmod +x ai-cv-summarize

# Test run
./ai-cv-summarize
```

## 🔄 Step 7: Setup Systemd Service (Production)

```bash
# Create systemd service file
sudo nano /etc/systemd/system/ai-cv-summarize.service
```

**Content:**
```ini
[Unit]
Description=AI CV Summarize Backend Service
After=network.target mongod.service redis-server.service
Requires=mongod.service redis-server.service

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/api/ai-summarize
ExecStart=/home/ubuntu/api/ai-summarize/ai-cv-summarize
Restart=always
RestartSec=5
StandardOutput=append:/var/log/ai-cv-summarize/output.log
StandardError=append:/var/log/ai-cv-summarize/error.log

# Environment file
EnvironmentFile=/home/ubuntu/api/ai-summarize/.env

[Install]
WantedBy=multi-user.target
```

**Setup logging:**
```bash
# Create log directory
sudo mkdir -p /var/log/ai-cv-summarize
sudo chown ubuntu:ubuntu /var/log/ai-cv-summarize

# Reload systemd
sudo systemctl daemon-reload

# Enable service
sudo systemctl enable ai-cv-summarize

# Start service
sudo systemctl start ai-cv-summarize

# Check status
sudo systemctl status ai-cv-summarize

# View logs
sudo journalctl -u ai-cv-summarize -f
```

## 🌐 Step 8: Setup Nginx (Reverse Proxy)

```bash
# Install Nginx
sudo apt install -y nginx

# Create Nginx configuration
sudo nano /etc/nginx/sites-available/ai-cv-summarize
```

**Content:**
```nginx
server {
    listen 80;
    server_name your-domain.com;  # Or your EC2 IP

    client_max_body_size 20M;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Timeouts for long-running AI requests
        proxy_connect_timeout 600;
        proxy_send_timeout 600;
        proxy_read_timeout 600;
        send_timeout 600;
    }
}
```

**Enable and start:**
```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/ai-cv-summarize /etc/nginx/sites-enabled/

# Remove default site
sudo rm /etc/nginx/sites-enabled/default

# Test configuration
sudo nginx -t

# Restart Nginx
sudo systemctl restart nginx
sudo systemctl enable nginx
```

## 🔒 Step 9: Setup SSL with Let's Encrypt (Optional)

```bash
# Install Certbot
sudo apt install -y certbot python3-certbot-nginx

# Get SSL certificate
sudo certbot --nginx -d your-domain.com

# Auto-renewal is configured automatically
```

## 🔥 Step 10: Configure Firewall

```bash
# Enable UFW
sudo ufw enable

# Allow SSH
sudo ufw allow 22/tcp

# Allow HTTP
sudo ufw allow 80/tcp

# Allow HTTPS (if using SSL)
sudo ufw allow 443/tcp

# Check status
sudo ufw status
```

**Or configure AWS Security Group:**
- Port 22 (SSH)
- Port 80 (HTTP)
- Port 443 (HTTPS)
- Port 8080 (API - optional, if direct access needed)

## 📊 Step 11: Monitoring & Management

### **Service Management:**
```bash
# Start service
sudo systemctl start ai-cv-summarize

# Stop service
sudo systemctl stop ai-cv-summarize

# Restart service
sudo systemctl restart ai-cv-summarize

# Check status
sudo systemctl status ai-cv-summarize

# View logs
sudo journalctl -u ai-cv-summarize -f
sudo tail -f /var/log/ai-cv-summarize/output.log
sudo tail -f /var/log/ai-cv-summarize/error.log
```

### **MongoDB Management:**
```bash
# Access MongoDB
mongosh

# Check databases
show dbs

# Use database
use ai_cv_summarize

# Check collections
show collections

# Check job descriptions
db.job_descriptions.find().pretty()
```

### **Redis Management:**
```bash
# Access Redis
redis-cli

# Check keys
KEYS *

# Monitor Redis
MONITOR
```

## 🧪 Step 12: Test Deployment

```bash
# Health check
curl http://your-ec2-ip-or-domain/health

# Upload files
curl -X POST http://your-ec2-ip-or-domain/api/v1/upload \
  -F "cv_file=@cv.pdf" \
  -F "project_file=@project.docx"

# Start evaluation
curl -X POST http://your-ec2-ip-or-domain/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{"cv_file": "filename.pdf", "project_file": "filename.docx"}'

# Check result
curl http://your-ec2-ip-or-domain/api/v1/result/{job_id}
```

## 🔧 Troubleshooting

### **If service fails to start:**
```bash
# Check logs
sudo journalctl -u ai-cv-summarize -n 50

# Check if ports are in use
sudo netstat -tlnp | grep 8080

# Check MongoDB connection
mongosh --eval "db.version()"

# Check Redis connection
redis-cli ping
```

### **If out of memory:**
```bash
# Add swap space
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

### **Performance Tuning:**
```bash
# Increase file upload limits in Nginx
sudo nano /etc/nginx/nginx.conf
# Add: client_max_body_size 20M;

# Optimize MongoDB
sudo nano /etc/mongod.conf
# Adjust memory limits if needed
```

## 📈 Monitoring & Logs

```bash
# Watch application logs
tail -f /var/log/ai-cv-summarize/output.log

# Watch error logs
tail -f /var/log/ai-cv-summarize/error.log

# System resource usage
htop

# Disk usage
df -h

# Memory usage
free -h
```

## ✅ Deployment Checklist

- [ ] Golang 1.21 installed
- [ ] MongoDB installed and running
- [ ] Redis installed and running
- [ ] Application cloned and built
- [ ] .env file configured with API keys
- [ ] Systemd service created and running
- [ ] Nginx configured and running
- [ ] Firewall/Security groups configured
- [ ] SSL certificate installed (optional)
- [ ] Application tested and working

## 🎯 Quick Start Script

Save this as `deploy.sh`:

```bash
#!/bin/bash

echo "🚀 Deploying AI CV Summarize to AWS"

# Install Golang
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install MongoDB
curl -fsSL https://www.mongodb.org/static/pgp/server-7.0.asc | sudo gpg -o /usr/share/keyrings/mongodb-server-7.0.gpg --dearmor
echo "deb [ arch=amd64,arm64 signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/7.0 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-7.0.list
sudo apt update
sudo apt install -y mongodb-org
sudo systemctl start mongod
sudo systemctl enable mongod

# Install Redis
sudo apt install -y redis-server
sudo systemctl start redis-server
sudo systemctl enable redis-server

# Build application
cd ~/api/ai-summarize
go mod download
go build -o ai-cv-summarize ./cmd/server/main.go

echo "✅ Installation complete!"
echo "🔑 Don't forget to configure .env file with your OpenAI API key"
```

Run it:
```bash
chmod +x deploy.sh
./deploy.sh
```

---

**Your application will be accessible at:**
- `http://your-ec2-ip:8080` (direct)
- `http://your-ec2-ip` (via Nginx)
- `https://your-domain.com` (with SSL)
