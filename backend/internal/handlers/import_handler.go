package handlers

import (
    "encoding/json"
    "net/http"

    "hyprlnk/internal/models"
    "hyprlnk/internal/services"
)

type ImportHandler struct {
    service services.HyprLinkService
}

func NewImportHandler(service services.HyprLinkService) *ImportHandler {
    return &ImportHandler{service: service}
}

func (h *ImportHandler) ImportBrowserData(w http.ResponseWriter, r *http.Request) {
    var importRequest struct {
        Bookmarks []models.ImportedBookmark `json:"bookmarks"`
        History   []models.HistoryEntry     `json:"history"`
        UseAI     bool                      `json:"use_ai"`
    }

    if err := json.NewDecoder(r.Body).Decode(&importRequest); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    importedCount, err := h.service.ImportBrowserData(importRequest.Bookmarks, importRequest.History, importRequest.UseAI)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    allBookmarks, _ := h.service.GetAllBookmarks()

    response := map[string]interface{}{
        "imported_count": importedCount,
        "total_count":    len(allBookmarks),
        "ai_processed":   importRequest.UseAI,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (h *ImportHandler) BulkSegmentBookmarks(w http.ResponseWriter, r *http.Request) {
    processedCount, err := h.service.BulkSegmentBookmarks()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    allBookmarks, _ := h.service.GetAllBookmarks()

    response := map[string]interface{}{
        "processed_count": processedCount,
        "total_count":     len(allBookmarks),
        "message":         "Bookmark segmentation completed",
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}