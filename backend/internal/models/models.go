package models

import "time"

type Bookmark struct {
    ID          int64     `json:"id"`
    URL         string    `json:"url"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Tags        []string  `json:"tags"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type Tab struct {
    URL        string `json:"url"`
    Title      string `json:"title"`
    Active     bool   `json:"active"`
    Index      int    `json:"index"`
    FavIconURL string `json:"favIconUrl"`
    Pinned     bool   `json:"pinned"`
}

type Session struct {
    ID          int64     `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Tabs        []Tab     `json:"tabs"`
    IsActive    bool      `json:"is_active"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type ImportedBookmark struct {
    URL       string    `json:"url"`
    Title     string    `json:"title"`
    Folder    string    `json:"folder"`
    AddedDate time.Time `json:"added_date"`
}

type HistoryEntry struct {
    URL           string    `json:"url"`
    Title         string    `json:"title"`
    VisitCount    int       `json:"visit_count"`
    LastVisitTime time.Time `json:"last_visit_time"`
    SourceURL     string    `json:"source_url,omitempty"`
    SourceTitle   string    `json:"source_title,omitempty"`
    LinkText      string    `json:"link_text,omitempty"`
}

type AISegmentation struct {
    Category    string   `json:"category"`
    Tags        []string `json:"tags"`
    Description string   `json:"description"`
    Confidence  float64  `json:"confidence"`
}

type LinkClick struct {
    ID               int64     `json:"id"`
    DestinationURL   string    `json:"destination_url"`
    DestinationTitle string    `json:"destination_title"`
    SourceURL        string    `json:"source_url"`
    SourceTitle      string    `json:"source_title"`
    LinkText         string    `json:"link_text"`
    ClickType        string    `json:"click_type"` // external_link, internal_link, form_submit
    Domain           string    `json:"domain"`
    IsNewTab         bool      `json:"is_new_tab"`
    Timestamp        time.Time `json:"timestamp"`
    CreatedAt        time.Time `json:"created_at"`
}