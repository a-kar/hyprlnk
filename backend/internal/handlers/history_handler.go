package handlers

import (
    "encoding/json"
    "net/http"

    "hyprlnk/internal/models"
    "hyprlnk/internal/services"
)

type HistoryHandler struct {
    service services.HyprLinkService
}

func NewHistoryHandler(service services.HyprLinkService) *HistoryHandler {
    return &HistoryHandler{service: service}
}

func (h *HistoryHandler) GetAll(w http.ResponseWriter, r *http.Request) {
    history, err := h.service.GetAllHistory()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(history)
}

func (h *HistoryHandler) GetToday(w http.ResponseWriter, r *http.Request) {
    history, err := h.service.GetTodaysHistory()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(history)
}

func (h *HistoryHandler) GetWeek(w http.ResponseWriter, r *http.Request) {
    history, err := h.service.GetWeekHistory()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(history)
}

func (h *HistoryHandler) GetMonth(w http.ResponseWriter, r *http.Request) {
    history, err := h.service.GetMonthHistory()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(history)
}

func (h *HistoryHandler) GetCount(w http.ResponseWriter, r *http.Request) {
    count, err := h.service.GetHistoryCount()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    response := map[string]interface{}{
        "total_count": count,
        "message":     "History count retrieved successfully",
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (h *HistoryHandler) Sync(w http.ResponseWriter, r *http.Request) {
    var historyRequest struct {
        History []models.HistoryEntry `json:"history"`
    }

    if err := json.NewDecoder(r.Body).Decode(&historyRequest); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    syncedCount, err := h.service.SyncHistory(historyRequest.History)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    totalCount, _ := h.service.GetHistoryCount()

    response := map[string]interface{}{
        "synced_count": syncedCount,
        "total_count":  totalCount,
        "message":      "History synchronized successfully",
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}