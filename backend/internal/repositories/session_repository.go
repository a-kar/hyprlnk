package repositories

import (
    "fmt"
    "time"

    "hyprlink/internal/models"
    "hyprlink/internal/storage"
)

type sessionRepository struct {
    storage *storage.ParquetStorage
}

func NewSessionRepository(storage *storage.ParquetStorage) SessionRepository {
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
    session.ID = time.Now().UnixNano()
    session.CreatedAt = time.Now()
    session.UpdatedAt = time.Now()
    session.IsActive = true

    sessions, err := r.storage.ReadSessions()
    if err != nil {
        return err
    }

    sessions = append(sessions, *session)
    return r.storage.WriteSessions(sessions)
}

func (r *sessionRepository) Update(session *models.Session) error {
    sessions, err := r.storage.ReadSessions()
    if err != nil {
        return err
    }

    for i, existing := range sessions {
        if existing.ID == session.ID {
            session.CreatedAt = existing.CreatedAt
            session.UpdatedAt = time.Now()
            sessions[i] = *session
            return r.storage.WriteSessions(sessions)
        }
    }

    return fmt.Errorf("session with ID %d not found", session.ID)
}

func (r *sessionRepository) Delete(id int64) error {
    sessions, err := r.storage.ReadSessions()
    if err != nil {
        return err
    }

    for i, session := range sessions {
        if session.ID == id {
            sessions = append(sessions[:i], sessions[i+1:]...)
            return r.storage.WriteSessions(sessions)
        }
    }

    return fmt.Errorf("session with ID %d not found", id)
}