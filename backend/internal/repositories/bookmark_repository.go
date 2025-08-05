package repositories

import (
    "fmt"
    "strings"
    "time"

    "hyprlink/internal/models"
    "hyprlink/internal/storage"
)

type bookmarkRepository struct {
    storage *storage.ParquetStorage
}

func NewBookmarkRepository(storage *storage.ParquetStorage) BookmarkRepository {
    return &bookmarkRepository{storage: storage}
}

func (r *bookmarkRepository) GetAll() ([]models.Bookmark, error) {
    return r.storage.ReadBookmarks()
}

func (r *bookmarkRepository) GetByID(id int64) (*models.Bookmark, error) {
    bookmarks, err := r.storage.ReadBookmarks()
    if err != nil {
        return nil, err
    }

    for _, bookmark := range bookmarks {
        if bookmark.ID == id {
            return &bookmark, nil
        }
    }

    return nil, fmt.Errorf("bookmark with ID %d not found", id)
}

func (r *bookmarkRepository) Create(bookmark *models.Bookmark) error {
    bookmark.ID = time.Now().UnixNano()
    bookmark.CreatedAt = time.Now()
    bookmark.UpdatedAt = time.Now()

    bookmarks, err := r.storage.ReadBookmarks()
    if err != nil {
        return err
    }

    bookmarks = append(bookmarks, *bookmark)
    return r.storage.WriteBookmarks(bookmarks)
}

func (r *bookmarkRepository) Update(bookmark *models.Bookmark) error {
    bookmarks, err := r.storage.ReadBookmarks()
    if err != nil {
        return err
    }

    for i, existing := range bookmarks {
        if existing.ID == bookmark.ID {
            bookmark.CreatedAt = existing.CreatedAt
            bookmark.UpdatedAt = time.Now()
            bookmarks[i] = *bookmark
            return r.storage.WriteBookmarks(bookmarks)
        }
    }

    return fmt.Errorf("bookmark with ID %d not found", bookmark.ID)
}

func (r *bookmarkRepository) Delete(id int64) error {
    bookmarks, err := r.storage.ReadBookmarks()
    if err != nil {
        return err
    }

    for i, bookmark := range bookmarks {
        if bookmark.ID == id {
            bookmarks = append(bookmarks[:i], bookmarks[i+1:]...)
            return r.storage.WriteBookmarks(bookmarks)
        }
    }

    return fmt.Errorf("bookmark with ID %d not found", id)
}

func (r *bookmarkRepository) Search(query string) ([]models.Bookmark, error) {
    bookmarks, err := r.storage.ReadBookmarks()
    if err != nil {
        return nil, err
    }

    var results []models.Bookmark
    queryLower := strings.ToLower(query)
    
    for _, bookmark := range bookmarks {
        if strings.Contains(strings.ToLower(bookmark.Title), queryLower) ||
           strings.Contains(strings.ToLower(bookmark.URL), queryLower) ||
           strings.Contains(strings.ToLower(bookmark.Description), queryLower) {
            results = append(results, bookmark)
            continue
        }
        
        for _, tag := range bookmark.Tags {
            if strings.Contains(strings.ToLower(tag), queryLower) {
                results = append(results, bookmark)
                break
            }
        }
    }

    return results, nil
}