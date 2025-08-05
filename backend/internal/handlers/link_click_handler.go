package handlers

import (
    "encoding/json"
    "net/http"
    "time"

    "hyprlink/internal/models"
    "hyprlink/internal/services"
)

type LinkClickHandler struct {
    service services.HyprLinkService
}

func NewLinkClickHandler(service services.HyprLinkService) *LinkClickHandler {
    return &LinkClickHandler{service: service}
}

func (h *LinkClickHandler) GetAll(w http.ResponseWriter, r *http.Request) {
    clicks, err := h.service.GetAllLinkClicks()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(clicks)
}

func (h *LinkClickHandler) Sync(w http.ResponseWriter, r *http.Request) {
    var clickRequest struct {
        Clicks []struct {
            DestinationURL   string `json:"destinationUrl"`
            DestinationTitle string `json:"destinationTitle"`
            SourceURL        string `json:"sourceUrl"`
            SourceTitle      string `json:"sourceTitle"`
            LinkText         string `json:"linkText"`
            ClickType        string `json:"clickType"`
            Domain           string `json:"domain"`
            IsNewTab         bool   `json:"isNewTab"`
            Timestamp        int64  `json:"timestamp"`
        } `json:"clicks"`
    }

    if err := json.NewDecoder(r.Body).Decode(&clickRequest); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    var clicks []models.LinkClick
    for _, click := range clickRequest.Clicks {
        newClick := models.LinkClick{
            DestinationURL:   click.DestinationURL,
            DestinationTitle: click.DestinationTitle,
            SourceURL:        click.SourceURL,
            SourceTitle:      click.SourceTitle,
            LinkText:         click.LinkText,
            ClickType:        click.ClickType,
            Domain:           click.Domain,
            IsNewTab:         click.IsNewTab,
            Timestamp:        time.UnixMilli(click.Timestamp),
        }
        clicks = append(clicks, newClick)
    }

    syncedCount, err := h.service.SyncLinkClicks(clicks)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    allClicks, _ := h.service.GetAllLinkClicks()

    response := map[string]interface{}{
        "synced_count": syncedCount,
        "total_count":  len(allClicks),
        "message":      "Link clicks synchronized successfully",
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}