package repositories

import (
    "fmt"

    "hyprlnk/internal/models"
    "hyprlnk/internal/storage"
)

type sessionRepository struct {
    storage *storage.AppendLogStorage
}

func NewSessionRepository(storage *storage.AppendLogStorage) SessionRepository {
    return &sessionRepository{storage: storage}
}

func (r *sessionRepository) GetAll() ([]models.Session, error) {
    return r.storage.ReadSessions()
}

func (r *sessionRepository) GetByID(id int64) (*models.Session, error) {
    sessions, err := r.storage.ReadSessions()
    if err != nil {
        return nil, err
    }

    for _, session := range sessions {
        if session.ID == id {
            return &session, nil
        }
    }

    return nil, fmt.Errorf("session with ID %d not found", id)
}

func (r *sessionRepository) Create(session *models.Session) error {
    session.IsActive = true
    return r.storage.AddSession(*session)
}

func (r *sessionRepository) Update(session *models.Session) error {
    // Check if session exists first
    existing, err := r.GetByID(session.ID)
    if err != nil {
        return fmt.Errorf("session with ID %d not found", session.ID)
    }
    
    // Preserve creation time
    session.CreatedAt = existing.CreatedAt
    
    return r.storage.UpdateSession(*session)
}

func (r *sessionRepository) Delete(id int64) error {
    // Check if session exists first
    _, err := r.GetByID(id)
    if err != nil {
        return fmt.Errorf("session with ID %d not found", id)
    }
    
    return r.storage.DeleteSession(id)
}