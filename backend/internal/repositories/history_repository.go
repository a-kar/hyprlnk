package repositories

import (
    "time"

    "hyprlnk/internal/models"
    "hyprlnk/internal/storage"
)

type historyRepository struct {
    storage *storage.AppendLogStorage
}

func NewHistoryRepository(storage *storage.AppendLogStorage) HistoryRepository {
    return &historyRepository{storage: storage}
}

func (r *historyRepository) GetAll() ([]models.HistoryEntry, error) {
    return r.storage.ReadHistory()
}

func (r *historyRepository) GetToday() ([]models.HistoryEntry, error) {
    history, err := r.storage.ReadHistory()
    if err != nil {
        return nil, err
    }

    today := time.Now().Truncate(24 * time.Hour)
    tomorrow := today.Add(24 * time.Hour)

    var todaysHistory []models.HistoryEntry
    for _, entry := range history {
        if entry.LastVisitTime.After(today) && entry.LastVisitTime.Before(tomorrow) {
            todaysHistory = append(todaysHistory, entry)
        }
    }

    return todaysHistory, nil
}

func (r *historyRepository) GetWeek() ([]models.HistoryEntry, error) {
    history, err := r.storage.ReadHistory()
    if err != nil {
        return nil, err
    }

    now := time.Now()
    weekAgo := now.AddDate(0, 0, -7)

    var weekHistory []models.HistoryEntry
    for _, entry := range history {
        if entry.LastVisitTime.After(weekAgo) {
            weekHistory = append(weekHistory, entry)
        }
    }

    return weekHistory, nil
}

func (r *historyRepository) GetMonth() ([]models.HistoryEntry, error) {
    history, err := r.storage.ReadHistory()
    if err != nil {
        return nil, err
    }

    now := time.Now()
    monthAgo := now.AddDate(0, 0, -30)

    var monthHistory []models.HistoryEntry
    for _, entry := range history {
        if entry.LastVisitTime.After(monthAgo) {
            monthHistory = append(monthHistory, entry)
        }
    }

    return monthHistory, nil
}

func (r *historyRepository) GetCount() (int, error) {
    history, err := r.storage.ReadHistory()
    if err != nil {
        return 0, err
    }
    return len(history), nil
}

func (r *historyRepository) Sync(entries []models.HistoryEntry) (int, error) {
    existingHistory, err := r.storage.ReadHistory()
    if err != nil {
        return 0, err
    }

    historyMap := make(map[string]models.HistoryEntry)
    for _, entry := range existingHistory {
        historyMap[entry.URL] = entry
    }

    syncedCount := 0
    for _, newEntry := range entries {
        if existing, exists := historyMap[newEntry.URL]; exists {
            if newEntry.LastVisitTime.After(existing.LastVisitTime) {
                historyMap[newEntry.URL] = newEntry
                syncedCount++
            }
        } else {
            historyMap[newEntry.URL] = newEntry
            syncedCount++
        }
    }

    var allHistory []models.HistoryEntry
    for _, entry := range historyMap {
        allHistory = append(allHistory, entry)
    }

    err = r.storage.WriteHistory(allHistory)
    return syncedCount, err
}

func (r *historyRepository) EnrichWithLinkClicks(history []models.HistoryEntry) ([]models.HistoryEntry, error) {
    linkClicks, err := r.storage.ReadLinkClicks()
    if err != nil {
        return history, nil
    }
    
    clickMap := make(map[string]models.LinkClick)
    for _, click := range linkClicks {
        if existing, exists := clickMap[click.DestinationURL]; !exists || click.Timestamp.After(existing.Timestamp) {
            clickMap[click.DestinationURL] = click
        }
    }
    
    enrichedHistory := make([]models.HistoryEntry, len(history))
    for i, entry := range history {
        enrichedHistory[i] = entry
        if click, exists := clickMap[entry.URL]; exists {
            clickDay := click.Timestamp.Truncate(24 * time.Hour)
            visitDay := entry.LastVisitTime.Truncate(24 * time.Hour)
            if clickDay.Equal(visitDay) {
                enrichedHistory[i].SourceURL = click.SourceURL
                enrichedHistory[i].SourceTitle = click.SourceTitle
                enrichedHistory[i].LinkText = click.LinkText
            }
        }
    }
    
    return enrichedHistory, nil
}