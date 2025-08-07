# 🔗 HyprLnk

**Smart Session Management & Bookmark Organization**

HyprLnk is an intelligent session management system that revolutionizes how you work with browser tabs. Save and restore entire browser sessions instantly, organize bookmarks into smart collections, and automatically track your browsing history - all with a beautiful, modern interface.

![License](https://img.shields.io/badge/license-BSL%201.1-blue.svg)
![Docker](https://img.shields.io/badge/docker-ready-blue.svg)
![Go](https://img.shields.io/badge/go-1.21+-blue.svg)
![Self-Hosted](https://img.shields.io/badge/self--hosted-ready-green.svg)

## ✨ Features

### 🗂️ Smart Session Management (Main Feature)
- **Save Sessions**: Capture all open tabs with one click
- **Instant Restore**: Recreate your entire browsing environment instantly
- **Session History**: Track and manage multiple saved sessions
- **Update Sessions**: Modify existing sessions with new tabs
- **Smart Restoration**: Intelligently replaces current tabs when restoring

### 📚 Bookmark Collections
- **Smart Collections**: Organize bookmarks into meaningful collections
- **Fuzzy Search**: Find bookmarks instantly with advanced search algorithm
- **Visual Organization**: Beautiful interface with color-coded collections
- **Import/Export**: Bring your existing bookmarks from any browser
- **Bulk Operations**: Manage multiple bookmarks efficiently

### 📊 Activity Tracking
- **Today's History**: Automatically track visited links
- **Usage Analytics**: Understand your browsing patterns
- **Quick Actions**: Save frequently visited sites instantly
- **Auto-Import**: Browser extension syncs history automatically

### 🎨 Modern Interface
- **Clean Design**: Beautiful, responsive interface with Alpine.js
- **Color-Coded**: Each section has its own theme:
  - 🔵 **Bookmarks** = Blue (organization)
  - 🟣 **Sessions** = Purple (workflow)
  - 🟢 **Collections** = Green (categorization)
  - 🟠 **Today** = Orange (daily activity)
- **Mobile Friendly**: Works perfectly on all devices
- **Fast Performance**: Optimized for speed and efficiency

## 🚀 Quick Start

### 🐳 Using Docker (Recommended)

**One-command setup:**
```bash
git clone https://github.com/a-kar/hyprlnk.git
cd hyprlnk
docker-compose up -d
```

**Access HyprLnk:**
- 🌐 Web Interface: http://localhost:4381
- 📊 API Health: http://localhost:8080/health

### 🏠 Homelab Setup with Portainer

Perfect for self-hosting enthusiasts! Deploy HyprLnk in your homelab with just a few clicks.

#### Method 1: Portainer Stacks (Recommended)

1. **Open Portainer** → Go to **Stacks** → **Add Stack**
2. **Name your stack**: `hyprlnk`
3. **Paste this Docker Compose configuration:**

```yaml
version: '3.8'

services:
  hyprlnk:
    build: ./backend
    container_name: hyprlnk-backend
    ports:
      - "8080:8080"
    volumes:
      - hyprlnk_data:/app/data
    environment:
      - PORT=8080
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s

  hyprlnk-frontend:
    build: ./frontend
    container_name: hyprlnk-frontend
    ports:
      - "4381:80"
    depends_on:
      - hyprlnk
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:80"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

volumes:
  hyprlnk_data:
    driver: local

networks:
  default:
    name: hyprlnk-network
```

4. **Deploy the stack** and wait for containers to start
5. **Access HyprLnk** at `http://YOUR_SERVER_IP:4381`

#### Method 2: Docker Run Commands

```bash
# Create network
docker network create hyprlnk-network

# Create volume
docker volume create hyprlnk_data

# Run backend
docker run -d \
  --name hyprlnk-backend \
  --network hyprlnk-network \
  -p 8080:8080 \
  -v hyprlnk_data:/app/data \
  -e PORT=8080 \
  --restart unless-stopped \
  hyprlnk-backend

# Run frontend
docker run -d \
  --name hyprlnk-frontend \
  --network hyprlnk-network \
  -p 4381:80 \
  --restart unless-stopped \
  hyprlnk-frontend
```

### 🔧 Custom Port Configuration

**Change the default port (4381):**

Edit your `docker-compose.yml`:
```yaml
services:
  hyprlnk-frontend:
    ports:
      - "YOUR_CUSTOM_PORT:80"  # e.g., "8080:80" for port 8080
```

**For Portainer users:** Simply change the port in the stack configuration before deploying.

## 🌐 Reverse Proxy Setup

### Nginx Proxy Manager (Homelab Favorite)
1. **Add Proxy Host** in Nginx Proxy Manager
2. **Domain Names**: `hyprlnk.yourdomain.com`
3. **Forward Hostname/IP**: `YOUR_SERVER_IP`
4. **Forward Port**: `4381`
5. **Enable SSL** with Let's Encrypt

### Traefik
```yaml
services:
  hyprlnk-frontend:
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.hyprlnk.rule=Host(`hyprlnk.yourdomain.com`)"
      - "traefik.http.routers.hyprlnk.tls.certresolver=letsencrypt"
      - "traefik.http.services.hyprlnk.loadbalancer.server.port=80"
```

### Caddy
```caddy
hyprlnk.yourdomain.com {
    reverse_proxy localhost:4381
}
```

### Manual Nginx
```nginx
server {
    listen 80;
    server_name hyprlnk.yourdomain.com;
    
    location / {
        proxy_pass http://localhost:4381;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 🔌 Browser Extension Setup

**For full functionality** (session management, history sync), install the browser extension:

### Chrome/Edge/Brave
1. Download the extension files or clone the repo
2. Open `chrome://extensions/`
3. Enable **"Developer mode"** (top right)
4. Click **"Load unpacked"**
5. Select the `frontend/extension` folder
6. Pin the extension to your toolbar

### Firefox
1. Open `about:debugging#/runtime/this-firefox`
2. Click **"Load Temporary Add-on"**
3. Select `frontend/extension/manifest.json`
4. Extension will be active until Firefox restart

### Extension Features
- 💾 **Save current page** as bookmark
- 📁 **Save entire session** with all tabs
- 🔄 **Update existing sessions**
- 📊 **Auto-sync browsing history**
- ⚡ **Right-click context menu** for quick actions

## 🗄️ Data Storage & Architecture

### Storage System
Hyprlink uses **Apache Parquet** for efficient, columnar data storage:

```
data/
├── bookmarks.parquet    # All saved bookmarks
├── sessions.parquet     # Browser sessions
└── history.parquet      # Browsing history
```

**Benefits of Parquet:**
- 🗜️ **Compression**: 50-80% smaller than JSON
- ⚡ **Fast Queries**: Columnar format optimized for reading
- 🔄 **Schema Evolution**: Easy to add new fields
- 🔍 **Analytics Ready**: Compatible with data analysis tools

### Architecture Overview
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Browser       │    │   Frontend       │    │   Backend       │
│   Extension     │◄──►│   (Static HTML)  │◄──►│   (Go API)      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 ▼
                    ┌─────────────────────────┐
                    │   Parquet Files         │
                    │   (Data Storage)        │
                    └─────────────────────────┘
```

## ⚙️ Configuration Options

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Backend API port |
| `DATA_DIR` | `/app/data` | Directory for Parquet files |
| `CORS_ORIGINS` | `*` | Allowed CORS origins |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |

### Docker Compose Overrides

Create `docker-compose.override.yml` for custom configurations:

```yaml
version: '3.8'

services:
  hyprlink:
    environment:
      - LOG_LEVEL=debug
    volumes:
      - /your/custom/path:/app/data
      
  hyprlink-frontend:
    ports:
      - "8080:80"  # Custom port
```

## 🔒 Security & Privacy

### 🛡️ Security Features
- **Input Validation**: All inputs sanitized and validated
- **CORS Protection**: Configurable origin restrictions
- **Health Checks**: Built-in monitoring endpoints
- **Container Security**: Non-root user execution
- **Rate Limiting**: API request throttling

### 🔐 Privacy First
- **Local Storage**: All data stays on your server
- **No Tracking**: Zero telemetry, analytics, or external calls
- **No Cloud**: Completely self-contained
- **Open Source**: Fully auditable code
- **Data Ownership**: You control your data 100%

## 📊 API Reference

### Bookmarks API
```http
GET    /api/bookmarks              # List all bookmarks
POST   /api/bookmarks              # Create bookmark
PUT    /api/bookmarks/{id}         # Update bookmark
DELETE /api/bookmarks/{id}         # Delete bookmark
GET    /api/bookmarks/search?q={}  # Search bookmarks
```

### Sessions API
```http
GET    /api/sessions               # List all sessions
POST   /api/sessions               # Create session
PUT    /api/sessions/{id}          # Update session
DELETE /api/sessions/{id}          # Delete session
```

### History API
```http
GET    /api/history                # List all history
GET    /api/history/today          # Get today's history
POST   /api/history/sync           # Sync browser history
```

### Import/Export API
```http
POST   /api/import/browser         # Import browser bookmarks
POST   /api/ai/segment             # AI categorization
```

### Health Check
```http
GET    /health                     # Application health status
```

## 🛠️ Development Setup

### Prerequisites
- **Go**: 1.21 or higher
- **Docker**: Latest version
- **Git**: For version control

### Local Development

1. **Clone and setup:**
   ```bash
   git clone https://github.com/a-kar/hyprlink.git
   cd hyprlink-project
   ```

2. **Backend development:**
   ```bash
   cd backend
   go mod download
   go run main.go
   # API available at http://localhost:8080
   ```

3. **Frontend development:**
   ```bash
   cd frontend/web
   python -m http.server 8081
   # Frontend available at http://localhost:8081
   ```

4. **Extension development:**
   - Load `frontend/extension` as unpacked extension
   - Modify files and reload extension to test changes

## 🐛 Troubleshooting

### Common Issues

**🔧 Port Already in Use**
```bash
# Find what's using the port
sudo lsof -i :4381
# Kill the process or change Hyprlink's port
```

**🔌 Extension Not Working**
- Ensure Hyprlink is running on localhost:4381
- Check extension is loaded in developer mode
- Verify permissions in manifest.json
- Reload extension after any changes

**💾 Data Not Persisting**
```bash
# Check Docker volume
docker volume inspect hyprlink_data
# Verify volume mount
docker inspect hyprlink-backend | grep Mounts -A 10
```

**🚫 CORS Errors**
- Check if frontend and backend ports match your setup
- Verify CORS_ORIGINS environment variable
- Ensure both services are running

**📁 Import Failures**
- Verify JSON format matches schema
- Check file size (must be < 10MB)
- Ensure proper encoding (UTF-8)

### Getting Help

- 📖 **Documentation**: Check the `/docs` folder
- 🐛 **Bug Reports**: Open an issue on GitHub
- 💬 **Discussions**: Join GitHub Discussions
- 📧 **Email**: Create issue for direct support

## 🚀 Performance Tips

### For Large Datasets
- **Regular Cleanup**: Remove old history entries periodically
- **Index Management**: Parquet files are self-optimizing
- **Memory Limits**: Set Docker memory limits if needed
- **SSD Storage**: Use SSD for data volume for best performance

### Docker Optimization
```yaml
services:
  hyprlink:
    deploy:
      resources:
        limits:
          memory: 512M
        reservations:
          memory: 256M
```

## 🗺️ Roadmap

- [ ] **📱 Mobile Apps**: Native iOS and Android applications
- [ ] **☁️ Cloud Backup**: Optional encrypted cloud storage
- [ ] **👥 Multi-User**: User accounts and permissions
- [ ] **🔄 Real-time Sync**: Live updates across devices
- [ ] **📊 Advanced Analytics**: Detailed usage insights
- [ ] **🔌 Plugin System**: Custom extensions and integrations
- [ ] **🌍 Internationalization**: Multiple language support
- [ ] **🎨 Themes**: Custom UI themes and dark mode

## 📄 License

This project is licensed under the **Business Source License 1.1** - see the [LICENSE](LICENSE) file for details.

After 3 years (August 1, 2028), the license will automatically convert to Apache License 2.0.

## 🙏 Acknowledgments

- **[Apache Arrow](https://arrow.apache.org/)**: Efficient columnar data storage
- **[Alpine.js](https://alpinejs.dev/)**: Lightweight reactive UI framework  
- **[Tailwind CSS](https://tailwindcss.com/)**: Utility-first CSS framework
- **[Font Awesome](https://fontawesome.com/)**: Comprehensive icon library
- **[Gorilla Mux](https://github.com/gorilla/mux)**: HTTP router for Go
- **Self-hosting Community**: For inspiring this project

---

<div align="center">

**⭐ Star this repo if you find it useful!**

**Perfect for homelabs, personal servers, and privacy-focused setups**

[Report Bug](https://github.com/a-kar/hyprlink/issues) • [Request Feature](https://github.com/a-kar/hyprlink/issues) • [Join Discussion](https://github.com/a-kar/hyprlink/discussions)

Made with ❤️ for the self-hosting community

</div>