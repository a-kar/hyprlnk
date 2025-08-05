package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/gorilla/mux"
    "github.com/rs/cors"

    "hyprlink/internal/handlers"
    "hyprlink/internal/repositories"
    "hyprlink/internal/services"
    "hyprlink/internal/storage"
)

type App struct {
    bookmarkHandler   *handlers.BookmarkHandler
    sessionHandler    *handlers.SessionHandler
    historyHandler    *handlers.HistoryHandler
    linkClickHandler  *handlers.LinkClickHandler
    importHandler     *handlers.ImportHandler
}

func NewApp(dataDir string) *App {
    parquetStorage := storage.NewParquetStorage(dataDir)

    bookmarkRepo := repositories.NewBookmarkRepository(parquetStorage)
    sessionRepo := repositories.NewSessionRepository(parquetStorage)
    historyRepo := repositories.NewHistoryRepository(parquetStorage)
    linkClickRepo := repositories.NewLinkClickRepository(parquetStorage)
    importRepo := repositories.NewImportRepository(parquetStorage)

    hyprLinkService := services.NewHyprLinkService(
        bookmarkRepo,
        sessionRepo,
        historyRepo,
        linkClickRepo,
        importRepo,
    )

    return &App{
        bookmarkHandler:  handlers.NewBookmarkHandler(hyprLinkService),
        sessionHandler:   handlers.NewSessionHandler(hyprLinkService),
        historyHandler:   handlers.NewHistoryHandler(hyprLinkService),
        linkClickHandler: handlers.NewLinkClickHandler(hyprLinkService),
        importHandler:    handlers.NewImportHandler(hyprLinkService),
    }
}

func (app *App) setupRoutes() *mux.Router {
    router := mux.NewRouter()

    router.HandleFunc("/api/bookmarks", app.bookmarkHandler.GetAll).Methods("GET")
    router.HandleFunc("/api/bookmarks", app.bookmarkHandler.Create).Methods("POST")
    router.HandleFunc("/api/bookmarks/{id}", app.bookmarkHandler.Update).Methods("PUT")
    router.HandleFunc("/api/bookmarks/{id}", app.bookmarkHandler.Delete).Methods("DELETE")
    router.HandleFunc("/api/bookmarks/search", app.bookmarkHandler.Search).Methods("GET")

    router.HandleFunc("/api/sessions", app.sessionHandler.GetAll).Methods("GET")
    router.HandleFunc("/api/sessions", app.sessionHandler.Create).Methods("POST")
    router.HandleFunc("/api/sessions/{id}", app.sessionHandler.Update).Methods("PUT")
    router.HandleFunc("/api/sessions/{id}", app.sessionHandler.Delete).Methods("DELETE")
    
    router.HandleFunc("/api/history", app.historyHandler.GetAll).Methods("GET")
    router.HandleFunc("/api/history/today", app.historyHandler.GetToday).Methods("GET")
    router.HandleFunc("/api/history/week", app.historyHandler.GetWeek).Methods("GET")
    router.HandleFunc("/api/history/month", app.historyHandler.GetMonth).Methods("GET")
    router.HandleFunc("/api/history/count", app.historyHandler.GetCount).Methods("GET")
    router.HandleFunc("/api/history/sync", app.historyHandler.Sync).Methods("POST")
    
    router.HandleFunc("/api/link-clicks", app.linkClickHandler.GetAll).Methods("GET")
    router.HandleFunc("/api/link-clicks/sync", app.linkClickHandler.Sync).Methods("POST")
    
    router.HandleFunc("/api/import/browser", app.importHandler.ImportBrowserData).Methods("POST")
    router.HandleFunc("/api/segment", app.importHandler.BulkSegmentBookmarks).Methods("POST")
    
    router.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{
            "status":    "ok", 
            "timestamp": time.Now().UTC().Format(time.RFC3339),
        })
    }).Methods("GET")
    
    router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
    }).Methods("GET")

    return router
}

func main() {
    dataDir := os.Getenv("DATA_DIR")
    if dataDir == "" {
        dataDir = "./data"
    }

    app := NewApp(dataDir)
    router := app.setupRoutes()

    c := cors.New(cors.Options{
        AllowedOrigins:   []string{"*"},
        AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"*"},
        AllowCredentials: false,
        MaxAge:           86400,
    })

    handler := c.Handler(router)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    fmt.Printf("🚀 HyprLnk API starting on port %s\n", port)
    log.Fatal(http.ListenAndServe(":"+port, handler))
}