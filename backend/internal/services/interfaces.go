package services

import "hyprlnk/internal/models"

type HyprLinkService interface {
    GetAllBookmarks() ([]models.Bookmark, error)
    CreateBookmark(bookmark *models.Bookmark) error
    UpdateBookmark(bookmark *models.Bookmark) error
    DeleteBookmark(id int64) error
    SearchBookmarks(query string) ([]models.Bookmark, error)
    
    GetAllSessions() ([]models.Session, error)
    CreateSession(session *models.Session) error
    UpdateSession(session *models.Session) error
    DeleteSession(id int64) error
    
    GetAllHistory() ([]models.HistoryEntry, error)
    GetTodaysHistory() ([]models.HistoryEntry, error)
    GetWeekHistory() ([]models.HistoryEntry, error)
    GetMonthHistory() ([]models.HistoryEntry, error)
    GetHistoryCount() (int, error)
    SyncHistory(entries []models.HistoryEntry) (int, error)
    
    GetAllLinkClicks() ([]models.LinkClick, error)
    SyncLinkClicks(clicks []models.LinkClick) (int, error)
    
    ImportBrowserData(bookmarks []models.ImportedBookmark, history []models.HistoryEntry, useAI bool) (int, error)
    BulkSegmentBookmarks() (int, error)
}