package repositories

import (
    "hyprlnk/internal/models"
)

type BookmarkRepository interface {
    GetAll() ([]models.Bookmark, error)
    GetByID(id int64) (*models.Bookmark, error)
    Create(bookmark *models.Bookmark) error
    Update(bookmark *models.Bookmark) error
    Delete(id int64) error
    Search(query string) ([]models.Bookmark, error)
}

type SessionRepository interface {
    GetAll() ([]models.Session, error)
    GetByID(id int64) (*models.Session, error)
    Create(session *models.Session) error
    Update(session *models.Session) error
    Delete(id int64) error
}

type HistoryRepository interface {
    GetAll() ([]models.HistoryEntry, error)
    GetToday() ([]models.HistoryEntry, error)
    GetWeek() ([]models.HistoryEntry, error)
    GetMonth() ([]models.HistoryEntry, error)
    GetCount() (int, error)
    Sync(entries []models.HistoryEntry) (int, error)
    EnrichWithLinkClicks(entries []models.HistoryEntry) ([]models.HistoryEntry, error)
}

type LinkClickRepository interface {
    GetAll() ([]models.LinkClick, error)
    Create(clicks []models.LinkClick) error
    Sync(clicks []models.LinkClick) (int, error)
}

type ImportRepository interface {
    ImportBrowserData(bookmarks []models.ImportedBookmark, history []models.HistoryEntry, useAI bool) (int, error)
}