package handlers

import (
    "encoding/json"
    "net/http"
    "strconv"

    "github.com/gorilla/mux"
    "hyprlink/internal/models"
    "hyprlink/internal/services"
)

type BookmarkHandler struct {
    service services.HyprLinkService
}

func NewBookmarkHandler(service services.HyprLinkService) *BookmarkHandler {
    return &BookmarkHandler{service: service}
}

func (h *BookmarkHandler) GetAll(w http.ResponseWriter, r *http.Request) {
    bookmarks, err := h.service.GetAllBookmarks()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(bookmarks)
}

func (h *BookmarkHandler) Create(w http.ResponseWriter, r *http.Request) {
    var bookmark models.Bookmark
    if err := json.NewDecoder(r.Body).Decode(&bookmark); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if err := h.service.CreateBookmark(&bookmark); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(bookmark)
}

func (h *BookmarkHandler) Update(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
        return
    }

    var bookmark models.Bookmark
    if err := json.NewDecoder(r.Body).Decode(&bookmark); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    bookmark.ID = id
    if err := h.service.UpdateBookmark(&bookmark); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(bookmark)
}

func (h *BookmarkHandler) Delete(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
        return
    }

    if err := h.service.DeleteBookmark(id); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

func (h *BookmarkHandler) Search(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    if query == "" {
        http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
        return
    }

    results, err := h.service.SearchBookmarks(query)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(results)
}