package repositories

import (
    "time"

    "hyprlnk/internal/models"
    "hyprlnk/internal/storage"
)

type linkClickRepository struct {
    storage *storage.AppendLogStorage
}

func NewLinkClickRepository(storage *storage.AppendLogStorage) LinkClickRepository {
    return &linkClickRepository{storage: storage}
}

func (r *linkClickRepository) GetAll() ([]models.LinkClick, error) {
    return r.storage.ReadLinkClicks()
}

func (r *linkClickRepository) Create(clicks []models.LinkClick) error {
    // Use batch write for multiple clicks
    for i := range clicks {
        if clicks[i].CreatedAt.IsZero() {
            clicks[i].CreatedAt = time.Now()
        }
    }
    return r.storage.WriteLinkClicks(clicks)
}

func (r *linkClickRepository) Sync(clicks []models.LinkClick) (int, error) {
    // Use batch write for sync operation
    for i := range clicks {
        if clicks[i].CreatedAt.IsZero() {
            clicks[i].CreatedAt = time.Now()
        }
    }
    
    err := r.storage.WriteLinkClicks(clicks)
    return len(clicks), err
}