package repositories

import (
    "fmt"
    "time"

    "hyprlnk/internal/models"
    "hyprlnk/internal/storage"
)

type importRepository struct {
    storage *storage.AppendLogStorage
}

func NewImportRepository(storage *storage.AppendLogStorage) ImportRepository {
    return &importRepository{storage: storage}
}

func (r *importRepository) ImportBrowserData(importedBookmarks []models.ImportedBookmark, history []models.HistoryEntry, useAI bool) (int, error) {
    existingBookmarks, err := r.storage.ReadBookmarks()
    if err != nil {
        return 0, err
    }

    var newBookmarks []models.Bookmark
    for _, imported := range importedBookmarks {
        bookmark := models.Bookmark{
            ID:          time.Now().UnixNano(),
            URL:         imported.URL,
            Title:       imported.Title,
            Description: fmt.Sprintf("Imported from %s", imported.Folder),
            Tags:        []string{imported.Folder},
            CreatedAt:   imported.AddedDate,
            UpdatedAt:   time.Now(),
        }
        newBookmarks = append(newBookmarks, bookmark)
    }

    allBookmarks := append(existingBookmarks, newBookmarks...)
    
    if err := r.storage.WriteBookmarks(allBookmarks); err != nil {
        return 0, err
    }

    return len(newBookmarks), nil
}