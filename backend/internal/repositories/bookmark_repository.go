package repositories

import (
    "fmt"
    "strings"

    "hyprlnk/internal/models"
    "hyprlnk/internal/storage"
)

type bookmarkRepository struct {
    storage *storage.AppendLogStorage
}

func NewBookmarkRepository(storage *storage.AppendLogStorage) BookmarkRepository {
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
    return r.storage.AddBookmark(*bookmark)
}

func (r *bookmarkRepository) Update(bookmark *models.Bookmark) error {
    // Check if bookmark exists first
    existing, err := r.GetByID(bookmark.ID)
    if err != nil {
        return fmt.Errorf("bookmark with ID %d not found", bookmark.ID)
    }
    
    // Preserve creation time
    bookmark.CreatedAt = existing.CreatedAt
    
    return r.storage.UpdateBookmark(*bookmark)
}

func (r *bookmarkRepository) Delete(id int64) error {
    // Check if bookmark exists first
    _, err := r.GetByID(id)
    if err != nil {
        return fmt.Errorf("bookmark with ID %d not found", id)
    }
    
    return r.storage.DeleteBookmark(id)
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