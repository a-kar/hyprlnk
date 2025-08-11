package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"hyprlnk/internal/models"
)

// AppendLogStorage implements an append-only log pattern with periodic compaction
// This provides efficient writes while maintaining data persistence across restarts
type AppendLogStorage struct {
	dataDir        string
	
	// Bookmark storage
	bookmarkMainFile   string
	bookmarkDeltaFile  string
	bookmarkDeltaBuffer []models.Bookmark
	bookmarkDeltaCount  int
	
	// Session storage  
	sessionMainFile    string
	sessionDeltaFile   string
	sessionDeltaBuffer []models.Session
	sessionDeltaCount  int
	
	// History storage (append-only by nature, no updates/deletes)
	historyMainFile    string
	historyDeltaFile   string
	historyDeltaBuffer []models.HistoryEntry
	historyDeltaCount  int
	
	// LinkClick storage (append-only by nature)
	linkClickMainFile   string
	linkClickDeltaFile  string
	linkClickDeltaBuffer []models.LinkClick
	linkClickDeltaCount  int
	
	compactThreshold int
	mutex           sync.RWMutex
	flushTicker     *time.Ticker
	stopChan        chan bool
	parquetStorage  *ParquetStorage // Reuse existing Parquet logic
}

// NewAppendLogStorage creates a new append-log based storage
func NewAppendLogStorage(dataDir string) *AppendLogStorage {
	os.MkdirAll(dataDir, 0755)
	
	als := &AppendLogStorage{
		dataDir:          dataDir,
		
		// Bookmark files
		bookmarkMainFile:  filepath.Join(dataDir, "bookmarks.parquet"),
		bookmarkDeltaFile: filepath.Join(dataDir, "bookmarks.delta.json"),
		bookmarkDeltaBuffer: make([]models.Bookmark, 0),
		
		// Session files
		sessionMainFile:   filepath.Join(dataDir, "sessions.parquet"),
		sessionDeltaFile:  filepath.Join(dataDir, "sessions.delta.json"),
		sessionDeltaBuffer: make([]models.Session, 0),
		
		// History files  
		historyMainFile:   filepath.Join(dataDir, "history.parquet"),
		historyDeltaFile:  filepath.Join(dataDir, "history.delta.json"),
		historyDeltaBuffer: make([]models.HistoryEntry, 0),
		
		// LinkClick files
		linkClickMainFile:  filepath.Join(dataDir, "link_clicks.parquet"),
		linkClickDeltaFile: filepath.Join(dataDir, "link_clicks.delta.json"),
		linkClickDeltaBuffer: make([]models.LinkClick, 0),
		
		compactThreshold: 100, // Compact after 100 delta entries per type
		stopChan:        make(chan bool),
		parquetStorage:  &ParquetStorage{dataDir: dataDir},
	}
	
	// Load any existing delta entries on startup
	als.loadAllDeltaFiles()
	
	// Start periodic flush (every 30 seconds or on shutdown)
	als.startPeriodicFlush()
	
	return als
}

// Close flushes pending changes and stops background tasks
func (als *AppendLogStorage) Close() error {
	als.mutex.Lock()
	defer als.mutex.Unlock()
	
	// Stop periodic flush
	if als.flushTicker != nil {
		als.flushTicker.Stop()
		als.flushTicker = nil
	}
	
	// Signal stop (non-blocking)
	select {
	case als.stopChan <- true:
	default:
	}
	
	// Final flush all
	als.flushAllDelta()
	
	return nil
}

// ============== BOOKMARK METHODS ==============

// WriteBookmarks replaces all bookmarks (used for imports)
func (als *AppendLogStorage) WriteBookmarks(bookmarks []models.Bookmark) error {
	als.mutex.Lock()
	defer als.mutex.Unlock()
	
	// For bulk replace, write directly to main Parquet file
	if err := als.parquetStorage.WriteBookmarks(bookmarks); err != nil {
		return err
	}
	
	// Clear delta since we just wrote everything fresh
	os.Remove(als.bookmarkDeltaFile)
	als.bookmarkDeltaBuffer = make([]models.Bookmark, 0)
	als.bookmarkDeltaCount = 0
	
	return nil
}

// ReadBookmarks reads all bookmarks with updates/deletes applied
func (als *AppendLogStorage) ReadBookmarks() ([]models.Bookmark, error) {
	als.mutex.RLock()
	defer als.mutex.RUnlock()
	
	// Read main Parquet file
	mainBookmarks, err := als.parquetStorage.ReadBookmarks()
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read main bookmarks: %w", err)
	}
	
	// Build map to handle updates and deletions
	bookmarkMap := make(map[int64]models.Bookmark)
	
	// Add main bookmarks
	for _, b := range mainBookmarks {
		bookmarkMap[b.ID] = b
	}
	
	// Apply delta changes (updates, deletes, inserts)
	for _, b := range als.bookmarkDeltaBuffer {
		if b.Title == "__DELETED__" {
			delete(bookmarkMap, b.ID)  // Remove deleted items
		} else {
			bookmarkMap[b.ID] = b  // Add new or update existing
		}
	}
	
	// Convert back to slice
	result := make([]models.Bookmark, 0, len(bookmarkMap))
	for _, b := range bookmarkMap {
		result = append(result, b)
	}
	
	return result, nil
}

// AddBookmark adds a single bookmark efficiently
func (als *AppendLogStorage) AddBookmark(bookmark models.Bookmark) error {
	als.mutex.Lock()
	defer als.mutex.Unlock()
	
	// Generate ID if not set
	if bookmark.ID == 0 {
		bookmark.ID = time.Now().UnixNano()
	}
	
	// Set timestamps
	now := time.Now()
	bookmark.CreatedAt = now
	bookmark.UpdatedAt = now
	
	// Add to in-memory buffer
	als.bookmarkDeltaBuffer = append(als.bookmarkDeltaBuffer, bookmark)
	
	// Persist to delta log immediately (append-only, fast)
	if err := als.appendBookmarkToDelta(bookmark); err != nil {
		return fmt.Errorf("failed to append bookmark to delta: %w", err)
	}
	
	als.bookmarkDeltaCount++
	
	// Check if compaction is needed
	if als.bookmarkDeltaCount >= als.compactThreshold {
		go als.compactBookmarks()
	}
	
	return nil
}

// UpdateBookmark updates an existing bookmark
func (als *AppendLogStorage) UpdateBookmark(bookmark models.Bookmark) error {
	als.mutex.Lock()
	defer als.mutex.Unlock()
	
	// Update timestamp
	bookmark.UpdatedAt = time.Now()
	
	// Add updated bookmark to delta (latest version wins during read)
	als.bookmarkDeltaBuffer = append(als.bookmarkDeltaBuffer, bookmark)
	
	if err := als.appendBookmarkToDelta(bookmark); err != nil {
		return fmt.Errorf("failed to append bookmark update: %w", err)
	}
	
	als.bookmarkDeltaCount++
	
	if als.bookmarkDeltaCount >= als.compactThreshold {
		go als.compactBookmarks()
	}
	
	return nil
}

// DeleteBookmark marks a bookmark as deleted
func (als *AppendLogStorage) DeleteBookmark(id int64) error {
	als.mutex.Lock()
	defer als.mutex.Unlock()
	
	// Create a tombstone entry
	tombstone := models.Bookmark{
		ID:        id,
		Title:     "__DELETED__",
		UpdatedAt: time.Now(),
	}
	
	als.bookmarkDeltaBuffer = append(als.bookmarkDeltaBuffer, tombstone)
	
	if err := als.appendBookmarkToDelta(tombstone); err != nil {
		return fmt.Errorf("failed to append bookmark deletion: %w", err)
	}
	
	als.bookmarkDeltaCount++
	
	if als.bookmarkDeltaCount >= als.compactThreshold {
		go als.compactBookmarks()
	}
	
	return nil
}

// ============== SESSION METHODS ==============

// WriteSessions replaces all sessions (used for imports)
func (als *AppendLogStorage) WriteSessions(sessions []models.Session) error {
	als.mutex.Lock()
	defer als.mutex.Unlock()
	
	// For bulk replace, write directly to main Parquet file
	if err := als.parquetStorage.WriteSessions(sessions); err != nil {
		return err
	}
	
	// Clear delta
	os.Remove(als.sessionDeltaFile)
	als.sessionDeltaBuffer = make([]models.Session, 0)
	als.sessionDeltaCount = 0
	
	return nil
}

// ReadSessions reads all sessions with updates/deletes applied
func (als *AppendLogStorage) ReadSessions() ([]models.Session, error) {
	als.mutex.RLock()
	defer als.mutex.RUnlock()
	
	// Read main Parquet file
	mainSessions, err := als.parquetStorage.ReadSessions()
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read main sessions: %w", err)
	}
	
	// Build map to handle updates and deletions
	sessionMap := make(map[int64]models.Session)
	
	// Add main sessions
	for _, s := range mainSessions {
		sessionMap[s.ID] = s
	}
	
	// Apply delta changes
	for _, s := range als.sessionDeltaBuffer {
		if s.Name == "__DELETED__" {
			delete(sessionMap, s.ID)
		} else {
			sessionMap[s.ID] = s
		}
	}
	
	// Convert back to slice
	result := make([]models.Session, 0, len(sessionMap))
	for _, s := range sessionMap {
		result = append(result, s)
	}
	
	return result, nil
}

// AddSession adds a single session
func (als *AppendLogStorage) AddSession(session models.Session) error {
	als.mutex.Lock()
	defer als.mutex.Unlock()
	
	// Generate ID if not set
	if session.ID == 0 {
		session.ID = time.Now().UnixNano()
	}
	
	// Set timestamps
	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now
	
	// Add to buffer
	als.sessionDeltaBuffer = append(als.sessionDeltaBuffer, session)
	
	// Persist to delta log
	if err := als.appendSessionToDelta(session); err != nil {
		return fmt.Errorf("failed to append session to delta: %w", err)
	}
	
	als.sessionDeltaCount++
	
	if als.sessionDeltaCount >= als.compactThreshold {
		go als.compactSessions()
	}
	
	return nil
}

// UpdateSession updates an existing session
func (als *AppendLogStorage) UpdateSession(session models.Session) error {
	als.mutex.Lock()
	defer als.mutex.Unlock()
	
	session.UpdatedAt = time.Now()
	
	als.sessionDeltaBuffer = append(als.sessionDeltaBuffer, session)
	
	if err := als.appendSessionToDelta(session); err != nil {
		return fmt.Errorf("failed to append session update: %w", err)
	}
	
	als.sessionDeltaCount++
	
	if als.sessionDeltaCount >= als.compactThreshold {
		go als.compactSessions()
	}
	
	return nil
}

// DeleteSession marks a session as deleted
func (als *AppendLogStorage) DeleteSession(id int64) error {
	als.mutex.Lock()
	defer als.mutex.Unlock()
	
	tombstone := models.Session{
		ID:        id,
		Name:      "__DELETED__",
		UpdatedAt: time.Now(),
	}
	
	als.sessionDeltaBuffer = append(als.sessionDeltaBuffer, tombstone)
	
	if err := als.appendSessionToDelta(tombstone); err != nil {
		return fmt.Errorf("failed to append session deletion: %w", err)
	}
	
	als.sessionDeltaCount++
	
	if als.sessionDeltaCount >= als.compactThreshold {
		go als.compactSessions()
	}
	
	return nil
}

// ============== HISTORY METHODS ==============
// History is append-only by nature (no updates/deletes)

// WriteHistory writes history entries (batch operation)
func (als *AppendLogStorage) WriteHistory(history []models.HistoryEntry) error {
	als.mutex.Lock()
	defer als.mutex.Unlock()
	
	// For batch operations, append to delta buffer
	for _, entry := range history {
		als.historyDeltaBuffer = append(als.historyDeltaBuffer, entry)
		
		if err := als.appendHistoryToDelta(entry); err != nil {
			return fmt.Errorf("failed to append history: %w", err)
		}
		
		als.historyDeltaCount++
	}
	
	// Compact if needed
	if als.historyDeltaCount >= als.compactThreshold {
		go als.compactHistory()
	}
	
	return nil
}

// ReadHistory reads all history entries
func (als *AppendLogStorage) ReadHistory() ([]models.HistoryEntry, error) {
	als.mutex.RLock()
	defer als.mutex.RUnlock()
	
	// Read main file
	mainHistory, err := als.parquetStorage.ReadHistory()
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read main history: %w", err)
	}
	
	// Append delta buffer (no deduplication needed for history)
	result := append(mainHistory, als.historyDeltaBuffer...)
	
	return result, nil
}

// ============== LINK CLICK METHODS ==============
// LinkClicks are append-only by nature

// WriteLinkClicks writes link click entries (batch operation)
func (als *AppendLogStorage) WriteLinkClicks(clicks []models.LinkClick) error {
	als.mutex.Lock()
	defer als.mutex.Unlock()
	
	// For batch operations, append to delta buffer
	for _, click := range clicks {
		// Generate ID if not set
		if click.ID == 0 {
			click.ID = time.Now().UnixNano()
		}
		
		als.linkClickDeltaBuffer = append(als.linkClickDeltaBuffer, click)
		
		if err := als.appendLinkClickToDelta(click); err != nil {
			return fmt.Errorf("failed to append link click: %w", err)
		}
		
		als.linkClickDeltaCount++
	}
	
	// Compact if needed
	if als.linkClickDeltaCount >= als.compactThreshold {
		go als.compactLinkClicks()
	}
	
	return nil
}

// ReadLinkClicks reads all link click entries
func (als *AppendLogStorage) ReadLinkClicks() ([]models.LinkClick, error) {
	als.mutex.RLock()
	defer als.mutex.RUnlock()
	
	// Read main file
	mainClicks, err := als.parquetStorage.ReadLinkClicks()
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read main link clicks: %w", err)
	}
	
	// Append delta buffer (no deduplication needed)
	result := append(mainClicks, als.linkClickDeltaBuffer...)
	
	return result, nil
}

// ============== PRIVATE HELPER METHODS ==============

func (als *AppendLogStorage) loadAllDeltaFiles() {
	als.loadBookmarkDelta()
	als.loadSessionDelta()
	als.loadHistoryDelta()
	als.loadLinkClickDelta()
}

func (als *AppendLogStorage) loadBookmarkDelta() error {
	data, err := os.ReadFile(als.bookmarkDeltaFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		
		var bookmark models.Bookmark
		if err := json.Unmarshal(line, &bookmark); err != nil {
			fmt.Printf("Warning: corrupted bookmark delta entry: %v\n", err)
			continue
		}
		
		als.bookmarkDeltaBuffer = append(als.bookmarkDeltaBuffer, bookmark)
		als.bookmarkDeltaCount++
	}
	
	return nil
}

func (als *AppendLogStorage) loadSessionDelta() error {
	data, err := os.ReadFile(als.sessionDeltaFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		
		var session models.Session
		if err := json.Unmarshal(line, &session); err != nil {
			fmt.Printf("Warning: corrupted session delta entry: %v\n", err)
			continue
		}
		
		als.sessionDeltaBuffer = append(als.sessionDeltaBuffer, session)
		als.sessionDeltaCount++
	}
	
	return nil
}

func (als *AppendLogStorage) loadHistoryDelta() error {
	data, err := os.ReadFile(als.historyDeltaFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		
		var entry models.HistoryEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			fmt.Printf("Warning: corrupted history delta entry: %v\n", err)
			continue
		}
		
		als.historyDeltaBuffer = append(als.historyDeltaBuffer, entry)
		als.historyDeltaCount++
	}
	
	return nil
}

func (als *AppendLogStorage) loadLinkClickDelta() error {
	data, err := os.ReadFile(als.linkClickDeltaFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		
		var click models.LinkClick
		if err := json.Unmarshal(line, &click); err != nil {
			fmt.Printf("Warning: corrupted link click delta entry: %v\n", err)
			continue
		}
		
		als.linkClickDeltaBuffer = append(als.linkClickDeltaBuffer, click)
		als.linkClickDeltaCount++
	}
	
	return nil
}

// Append methods for each type

func (als *AppendLogStorage) appendBookmarkToDelta(bookmark models.Bookmark) error {
	return als.appendToDeltaFile(als.bookmarkDeltaFile, bookmark)
}

func (als *AppendLogStorage) appendSessionToDelta(session models.Session) error {
	return als.appendToDeltaFile(als.sessionDeltaFile, session)
}

func (als *AppendLogStorage) appendHistoryToDelta(entry models.HistoryEntry) error {
	return als.appendToDeltaFile(als.historyDeltaFile, entry)
}

func (als *AppendLogStorage) appendLinkClickToDelta(click models.LinkClick) error {
	return als.appendToDeltaFile(als.linkClickDeltaFile, click)
}

func (als *AppendLogStorage) appendToDeltaFile(filename string, data interface{}) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	
	jsonData = append(jsonData, '\n')
	
	if _, err := file.Write(jsonData); err != nil {
		return err
	}
	
	return file.Sync()
}

// Compaction methods

func (als *AppendLogStorage) compactBookmarks() error {
	als.mutex.Lock()
	defer als.mutex.Unlock()
	
	// Read all current bookmarks (with deletes/updates applied)
	allBookmarks, _ := als.ReadBookmarks()
	
	// Write new main Parquet file
	if err := als.parquetStorage.WriteBookmarks(allBookmarks); err != nil {
		return fmt.Errorf("bookmark compaction failed: %w", err)
	}
	
	// Clear delta
	os.Remove(als.bookmarkDeltaFile)
	als.bookmarkDeltaBuffer = make([]models.Bookmark, 0)
	als.bookmarkDeltaCount = 0
	
	return nil
}

func (als *AppendLogStorage) compactSessions() error {
	als.mutex.Lock()
	defer als.mutex.Unlock()
	
	// Read all current sessions (with deletes/updates applied)
	allSessions, _ := als.ReadSessions()
	
	// Write new main Parquet file
	if err := als.parquetStorage.WriteSessions(allSessions); err != nil {
		return fmt.Errorf("session compaction failed: %w", err)
	}
	
	// Clear delta
	os.Remove(als.sessionDeltaFile)
	als.sessionDeltaBuffer = make([]models.Session, 0)
	als.sessionDeltaCount = 0
	
	return nil
}

func (als *AppendLogStorage) compactHistory() error {
	als.mutex.Lock()
	defer als.mutex.Unlock()
	
	// Read all history directly (already have lock, no deduplication needed)
	mainHistory, err := als.parquetStorage.ReadHistory()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read main history: %w", err)
	}
	
	// Append delta buffer (no deduplication needed for history)
	allHistory := append(mainHistory, als.historyDeltaBuffer...)
	
	// Write new main Parquet file
	if err := als.parquetStorage.WriteHistory(allHistory); err != nil {
		return fmt.Errorf("history compaction failed: %w", err)
	}
	
	// Clear delta
	os.Remove(als.historyDeltaFile)
	als.historyDeltaBuffer = make([]models.HistoryEntry, 0)
	als.historyDeltaCount = 0
	
	return nil
}

func (als *AppendLogStorage) compactLinkClicks() error {
	als.mutex.Lock()
	defer als.mutex.Unlock()
	
	// Read all link clicks directly (already have lock, no deduplication needed)
	mainClicks, err := als.parquetStorage.ReadLinkClicks()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read main link clicks: %w", err)
	}
	
	// Append delta buffer (no deduplication needed for link clicks)
	allClicks := append(mainClicks, als.linkClickDeltaBuffer...)
	
	// Write new main Parquet file
	if err := als.parquetStorage.WriteLinkClicks(allClicks); err != nil {
		return fmt.Errorf("link click compaction failed: %w", err)
	}
	
	// Clear delta
	os.Remove(als.linkClickDeltaFile)
	als.linkClickDeltaBuffer = make([]models.LinkClick, 0)
	als.linkClickDeltaCount = 0
	
	return nil
}

// Periodic flush

func (als *AppendLogStorage) startPeriodicFlush() {
	als.flushTicker = time.NewTicker(30 * time.Second)
	
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Graceful exit if ticker is nil
				return
			}
		}()
		
		for {
			select {
			case <-als.flushTicker.C:
				als.flushAllDelta()
			case <-als.stopChan:
				return
			}
		}
	}()
}

func (als *AppendLogStorage) flushAllDelta() {
	// Force sync all delta files to disk
	files := []string{
		als.bookmarkDeltaFile,
		als.sessionDeltaFile,
		als.historyDeltaFile,
		als.linkClickDeltaFile,
	}
	
	for _, filename := range files {
		if _, err := os.Stat(filename); err == nil {
			if file, err := os.OpenFile(filename, os.O_RDWR, 0644); err == nil {
				file.Sync()
				file.Close()
			}
		}
	}
}

// Helper function to split byte slice by newlines
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	
	return lines
}