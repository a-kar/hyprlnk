# 🔖 HyprLnk - Smart Bookmark Management

> **AI-powered bookmark organization with session management**

A modern bookmark management system with intelligent organization, session recovery, and cross-device synchronization.

## ✨ Features

- **Smart Organization**: AI-powered categorization and tagging
- **Session Management**: Save and restore browser sessions
- **Advanced Search**: Full-text search across bookmarks, URLs, and tags
- **Browser Extension**: Save bookmarks directly from any webpage
- **Import/Export**: Support for standard bookmark formats
- **Real-time Sync**: Keep bookmarks synchronized across devices
- **Modern UI**: Clean, responsive interface built with Tailwind CSS

## 🚀 Quick Start

### Using Docker (Recommended)

```bash
# Clone the repository
git clone <your-repo-url>
cd hyprlink-project

# Start with Docker Compose
docker-compose up -d

# Access the application
open http://localhost:3000
```

### Manual Setup

#### Backend Setup

```bash
cd backend

# Install Go dependencies
go mod download

# Run the server
go run main.go
```

#### Frontend Setup

The frontend is a static HTML application that runs directly in the browser. Simply serve the `frontend/web` directory with any web server.

#### Browser Extension

1. Open Chrome and navigate to `chrome://extensions/`
2. Enable "Developer mode"
3. Click "Load unpacked" and select the `frontend/extension` directory
4. The extension will appear in your toolbar

## 🔧 Configuration

Copy `.env.example` to `.env` and adjust settings:

```bash
cp .env.example .env
```

### Environment Variables

- `PORT`: Server port (default: 8080)
- `DATA_DIR`: Data storage directory (default: ./data)
- `AI_ENABLED`: Enable AI features (default: false)
- `CORS_ORIGINS`: Allowed CORS origins (default: *)

## 📚 API Documentation

### Bookmarks

- `GET /api/bookmarks` - Get all bookmarks
- `POST /api/bookmarks` - Create a bookmark
- `PUT /api/bookmarks/{id}` - Update a bookmark
- `DELETE /api/bookmarks/{id}` - Delete a bookmark
- `GET /api/bookmarks/search?q={query}` - Search bookmarks

### Sessions

- `GET /api/sessions` - Get all sessions
- `POST /api/sessions` - Save a session

### Import/Export

- `POST /api/import/browser` - Import browser bookmarks
- `POST /api/ai/segment` - AI-powered bookmark categorization

## 🏗️ Architecture

- **Backend**: Go with Gorilla Mux, Apache Arrow/Parquet for storage
- **Frontend**: Vanilla JavaScript with Alpine.js and Tailwind CSS
- **Storage**: Parquet files for efficient data storage and querying
- **Extension**: Chrome Extension Manifest V3

## 🔍 Data Storage

HyprLnk uses Apache Parquet files for efficient storage:

- `data/bookmarks.parquet` - Bookmark data
- `data/sessions.parquet` - Session data

Parquet provides:
- Columnar storage for fast queries
- Compression for space efficiency
- Schema evolution support
- Cross-language compatibility

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License.

## 🆘 Support

If you encounter any issues or have questions:

1. Check the [Issues](../../issues) page
2. Create a new issue with detailed information
3. Include your environment details and error messages

---

**Built with ❤️ using Go, JavaScript, and modern web technologies.**