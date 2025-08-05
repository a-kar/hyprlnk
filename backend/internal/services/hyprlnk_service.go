package services

import (
    "strings"
    "time"

    "hyprlink/internal/models"
    "hyprlink/internal/repositories"
)

type hyprLinkService struct {
    bookmarkRepo   repositories.BookmarkRepository
    sessionRepo    repositories.SessionRepository
    historyRepo    repositories.HistoryRepository
    linkClickRepo  repositories.LinkClickRepository
    importRepo     repositories.ImportRepository
}

func NewHyprLinkService(
    bookmarkRepo repositories.BookmarkRepository,
    sessionRepo repositories.SessionRepository,
    historyRepo repositories.HistoryRepository,
    linkClickRepo repositories.LinkClickRepository,
    importRepo repositories.ImportRepository,
) HyprLinkService {
    return &hyprLinkService{
        bookmarkRepo:  bookmarkRepo,
        sessionRepo:   sessionRepo,
        historyRepo:   historyRepo,
        linkClickRepo: linkClickRepo,
        importRepo:    importRepo,
    }
}

func (s *hyprLinkService) GetAllBookmarks() ([]models.Bookmark, error) {
    return s.bookmarkRepo.GetAll()
}

func (s *hyprLinkService) CreateBookmark(bookmark *models.Bookmark) error {
    return s.bookmarkRepo.Create(bookmark)
}

func (s *hyprLinkService) UpdateBookmark(bookmark *models.Bookmark) error {
    return s.bookmarkRepo.Update(bookmark)
}

func (s *hyprLinkService) DeleteBookmark(id int64) error {
    return s.bookmarkRepo.Delete(id)
}

func (s *hyprLinkService) SearchBookmarks(query string) ([]models.Bookmark, error) {
    return s.bookmarkRepo.Search(query)
}

func (s *hyprLinkService) GetAllSessions() ([]models.Session, error) {
    return s.sessionRepo.GetAll()
}

func (s *hyprLinkService) CreateSession(session *models.Session) error {
    return s.sessionRepo.Create(session)
}

func (s *hyprLinkService) UpdateSession(session *models.Session) error {
    return s.sessionRepo.Update(session)
}

func (s *hyprLinkService) DeleteSession(id int64) error {
    return s.sessionRepo.Delete(id)
}

func (s *hyprLinkService) GetAllHistory() ([]models.HistoryEntry, error) {
    history, err := s.historyRepo.GetAll()
    if err != nil {
        return nil, err
    }
    return s.historyRepo.EnrichWithLinkClicks(history)
}

func (s *hyprLinkService) GetTodaysHistory() ([]models.HistoryEntry, error) {
    history, err := s.historyRepo.GetToday()
    if err != nil {
        return nil, err
    }
    return s.historyRepo.EnrichWithLinkClicks(history)
}

func (s *hyprLinkService) GetWeekHistory() ([]models.HistoryEntry, error) {
    history, err := s.historyRepo.GetWeek()
    if err != nil {
        return nil, err
    }
    return s.historyRepo.EnrichWithLinkClicks(history)
}

func (s *hyprLinkService) GetMonthHistory() ([]models.HistoryEntry, error) {
    history, err := s.historyRepo.GetMonth()
    if err != nil {
        return nil, err
    }
    return s.historyRepo.EnrichWithLinkClicks(history)
}

func (s *hyprLinkService) GetHistoryCount() (int, error) {
    return s.historyRepo.GetCount()
}

func (s *hyprLinkService) SyncHistory(entries []models.HistoryEntry) (int, error) {
    return s.historyRepo.Sync(entries)
}

func (s *hyprLinkService) GetAllLinkClicks() ([]models.LinkClick, error) {
    return s.linkClickRepo.GetAll()
}

func (s *hyprLinkService) SyncLinkClicks(clicks []models.LinkClick) (int, error) {
    return s.linkClickRepo.Sync(clicks)
}

func (s *hyprLinkService) ImportBrowserData(bookmarks []models.ImportedBookmark, history []models.HistoryEntry, useAI bool) (int, error) {
    return s.importRepo.ImportBrowserData(bookmarks, history, useAI)
}

func (s *hyprLinkService) BulkSegmentBookmarks() (int, error) {
    bookmarks, err := s.bookmarkRepo.GetAll()
    if err != nil {
        return 0, err
    }

    processedCount := 0
    for _, bookmark := range bookmarks {
        if len(bookmark.Tags) == 0 {
            bookmark.Tags = s.generateTagsFromContent(bookmark)
            bookmark.UpdatedAt = time.Now()
            if err := s.bookmarkRepo.Update(&bookmark); err != nil {
                return processedCount, err
            }
            processedCount++
        }
    }

    return processedCount, nil
}

func (s *hyprLinkService) generateTagsFromContent(bookmark models.Bookmark) []string {
    tags := []string{}
    
    content := strings.ToLower(bookmark.Title + " " + bookmark.Description + " " + bookmark.URL)
    
    keywords := map[string]string{
        "github.com":     "development",
        "stackoverflow":  "development",
        "youtube.com":    "video",
        "medium.com":     "article",
        "news":           "news",
        "blog":           "blog",
        "tutorial":       "learning",
        "documentation":  "docs",
        "api":            "development",
        "react":          "frontend",
        "javascript":     "development",
        "python":         "development",
        "golang":         "development",
        "design":         "design",
        "tool":           "tools",
    }
    
    for keyword, tag := range keywords {
        if strings.Contains(content, keyword) {
            tags = append(tags, tag)
        }
    }
    
    if len(tags) == 0 {
        tags = append(tags, "uncategorized")
    }
    
    return tags
}