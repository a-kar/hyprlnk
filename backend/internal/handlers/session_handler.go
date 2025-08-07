package handlers

import (
    "encoding/json"
    "net/http"
    "strconv"

    "github.com/gorilla/mux"
    "hyprlnk/internal/models"
    "hyprlnk/internal/services"
)

type SessionHandler struct {
    service services.HyprLinkService
}

func NewSessionHandler(service services.HyprLinkService) *SessionHandler {
    return &SessionHandler{service: service}
}

func (h *SessionHandler) GetAll(w http.ResponseWriter, r *http.Request) {
    sessions, err := h.service.GetAllSessions()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(sessions)
}

func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
    var session models.Session
    if err := json.NewDecoder(r.Body).Decode(&session); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if err := h.service.CreateSession(&session); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(session)
}

func (h *SessionHandler) Update(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, "Invalid session ID", http.StatusBadRequest)
        return
    }

    var session models.Session
    if err := json.NewDecoder(r.Body).Decode(&session); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    session.ID = id
    if err := h.service.UpdateSession(&session); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(session)
}

func (h *SessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, "Invalid session ID", http.StatusBadRequest)
        return
    }

    if err := h.service.DeleteSession(id); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}