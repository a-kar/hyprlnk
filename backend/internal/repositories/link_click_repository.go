package repositories

import (
    "time"

    "hyprlink/internal/models"
    "hyprlink/internal/storage"
)

type linkClickRepository struct {
    storage *storage.ParquetStorage
}

func NewLinkClickRepository(storage *storage.ParquetStorage) LinkClickRepository {
    return &linkClickRepository{storage: storage}
}

func (r *linkClickRepository) GetAll() ([]models.LinkClick, error) {
    return r.storage.ReadLinkClicks()
}

func (r *linkClickRepository) Create(clicks []models.LinkClick) error {
    for i := range clicks {
        clicks[i].ID = time.Now().UnixNano()
        clicks[i].CreatedAt = time.Now()
    }

    existingClicks, err := r.storage.ReadLinkClicks()
    if err != nil {
        return err
    }

    allClicks := append(existingClicks, clicks...)
    return r.storage.WriteLinkClicks(allClicks)
}

func (r *linkClickRepository) Sync(clicks []models.LinkClick) (int, error) {
    for i := range clicks {
        clicks[i].ID = time.Now().UnixNano()
        clicks[i].CreatedAt = time.Now()
    }

    existingClicks, err := r.storage.ReadLinkClicks()
    if err != nil {
        return 0, err
    }

    allClicks := append(existingClicks, clicks...)
    err = r.storage.WriteLinkClicks(allClicks)
    return len(clicks), err
}