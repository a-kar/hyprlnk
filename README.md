# HyprLnk

**Self-hosted session management and browsing analytics for developers**

Save browser sessions, track browsing history, and analyze your web activity with a privacy-first approach.

![License](https://img.shields.io/badge/license-BSL%201.1-blue.svg)
![Docker](https://img.shields.io/badge/docker-ready-blue.svg)
![Go](https://img.shields.io/badge/go-1.21+-blue.svg)

**[📖 Documentation](https://hyprlnk.app) • [🚀 Releases](https://github.com/a-kar/hyprlnk/releases) • [🐛 Issues](https://github.com/a-kar/hyprlnk/issues)**

## Features

- **Session Management**: Save and restore entire browser sessions instantly
- **Browsing Analytics**: Track history with views by today, week, month, or all-time
- **Smart Search**: Fuzzy search across all your browsing history and bookmarks
- **Link Click Tracking**: Build complete navigation graphs of your web activity
- **File-Based Storage**: No database needed - JSON + Parquet for analytics
- **Privacy First**: All data stays on your server, zero external calls

## Quick Start

### Docker (Recommended)

```bash
git clone https://github.com/a-kar/hyprlnk.git
cd hyprlnk
docker-compose up -d
```

Access at: http://localhost:4381

### Browser Extension Setup

For full functionality, install the browser extension:

**Chrome/Edge/Brave:**
1. Go to `chrome://extensions/`
2. Enable "Developer mode"
3. Click "Load unpacked" 
4. Select the `frontend/extension` folder

**Firefox:**
1. Go to `about:debugging#/runtime/this-firefox`
2. Click "Load Temporary Add-on"
3. Select `frontend/extension/manifest.json`

## Architecture

- **Backend**: Go with Gorilla Mux
- **Frontend**: HTMX + Alpine.js + TailwindCSS
- **Storage**: JSON delta files + Parquet compaction
- **Extension**: Manifest V3 with full session capture

## Configuration

Environment variables:

```bash
PORT=8080                    # Backend API port
DATA_DIR=/app/data          # Storage directory
```

Runs on port 4381 by default.

## API Endpoints

```
GET    /api/sessions          # List sessions
POST   /api/sessions          # Create session
GET    /api/bookmarks         # List bookmarks  
GET    /api/history           # All history
GET    /api/history/today     # Today's history
GET    /health                # Health check
```

## Data Storage

Your data is stored as files:
```
data/
├── bookmarks.delta.json     # Real-time bookmarks
├── sessions.delta.json      # Real-time sessions  
├── history.delta.json       # Real-time history
├── bookmarks.parquet        # Analytics-ready format
├── sessions.parquet         # Analytics-ready format
└── history.parquet          # Analytics-ready format
```


## Development

```bash
# Backend
cd backend && go run main.go

# Frontend  
cd frontend/web && python -m http.server 8080
```

## License

Business Source License 1.1 - converts to Apache 2.0 on August 1, 2028.

---

**[Documentation](https://a-kar.github.io/hyprlnk/) • [Issues](https://github.com/a-kar/hyprlnk/issues) • [Releases](https://github.com/a-kar/hyprlnk/releases)**