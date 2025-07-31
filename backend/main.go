package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"

    "github.com/apache/arrow/go/v14/arrow"
    "github.com/apache/arrow/go/v14/arrow/array"
    "github.com/apache/arrow/go/v14/arrow/memory"
    "github.com/apache/arrow/go/v14/parquet"
    "github.com/apache/arrow/go/v14/parquet/file"
    "github.com/apache/arrow/go/v14/parquet/pqarrow"
    "github.com/gorilla/mux"
    "github.com/rs/cors"
)

type Bookmark struct {
    ID          int64     `json:"id"`
    URL         string    `json:"url"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Tags        []string  `json:"tags"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type Tab struct {
    URL       string `json:"url"`
    Title     string `json:"title"`
    Active    bool   `json:"active"`
    Index     int    `json:"index"`
    FavIconURL string `json:"favIconUrl"`
    Pinned    bool   `json:"pinned"`
}

type Session struct {
    ID          int64     `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Tabs        []Tab     `json:"tabs"`
    IsActive    bool      `json:"is_active"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type ImportedBookmark struct {
    URL       string    `json:"url"`
    Title     string    `json:"title"`
    Folder    string    `json:"folder"`
    AddedDate time.Time `json:"added_date"`
}

type HistoryEntry struct {
    URL           string    `json:"url"`
    Title         string    `json:"title"`
    VisitCount    int       `json:"visit_count"`
    LastVisitTime time.Time `json:"last_visit_time"`
    SourceURL     string    `json:"source_url,omitempty"`
    SourceTitle   string    `json:"source_title,omitempty"`
    LinkText      string    `json:"link_text,omitempty"`
}

type AISegmentation struct {
    Category    string   `json:"category"`
    Tags        []string `json:"tags"`
    Description string   `json:"description"`
    Confidence  float64  `json:"confidence"`
}

type LinkClick struct {
    ID              int64     `json:"id"`
    DestinationURL  string    `json:"destination_url"`
    DestinationTitle string   `json:"destination_title"`
    SourceURL       string    `json:"source_url"`
    SourceTitle     string    `json:"source_title"`
    LinkText        string    `json:"link_text"`
    ClickType       string    `json:"click_type"` // external_link, internal_link, form_submit
    Domain          string    `json:"domain"`
    IsNewTab        bool      `json:"is_new_tab"`
    Timestamp       time.Time `json:"timestamp"`
    CreatedAt       time.Time `json:"created_at"`
}

type BookmarkService struct {
    dataDir string
}

func NewBookmarkService(dataDir string) *BookmarkService {
    os.MkdirAll(dataDir, 0755)
    return &BookmarkService{dataDir: dataDir}
}

func (bs *BookmarkService) getBookmarkSchema() *arrow.Schema {
    return arrow.NewSchema([]arrow.Field{
        {Name: "id", Type: arrow.PrimitiveTypes.Int64},
        {Name: "url", Type: arrow.BinaryTypes.String},
        {Name: "title", Type: arrow.BinaryTypes.String},
        {Name: "description", Type: arrow.BinaryTypes.String},
        {Name: "tags", Type: arrow.BinaryTypes.String},
        {Name: "created_at", Type: arrow.FixedWidthTypes.Timestamp_ms},
        {Name: "updated_at", Type: arrow.FixedWidthTypes.Timestamp_ms},
    }, nil)
}

func (bs *BookmarkService) getSessionSchema() *arrow.Schema {
    return arrow.NewSchema([]arrow.Field{
        {Name: "id", Type: arrow.PrimitiveTypes.Int64},
        {Name: "name", Type: arrow.BinaryTypes.String},
        {Name: "description", Type: arrow.BinaryTypes.String},
        {Name: "tabs", Type: arrow.BinaryTypes.String},
        {Name: "is_active", Type: arrow.FixedWidthTypes.Boolean},
        {Name: "created_at", Type: arrow.FixedWidthTypes.Timestamp_ms},
        {Name: "updated_at", Type: arrow.FixedWidthTypes.Timestamp_ms},
    }, nil)
}

func (bs *BookmarkService) getHistorySchema() *arrow.Schema {
    return arrow.NewSchema([]arrow.Field{
        {Name: "url", Type: arrow.BinaryTypes.String},
        {Name: "title", Type: arrow.BinaryTypes.String},
        {Name: "visit_count", Type: arrow.PrimitiveTypes.Int32},
        {Name: "last_visit_time", Type: arrow.FixedWidthTypes.Timestamp_ms},
    }, nil)
}

func (bs *BookmarkService) getLinkClickSchema() *arrow.Schema {
    return arrow.NewSchema([]arrow.Field{
        {Name: "id", Type: arrow.PrimitiveTypes.Int64},
        {Name: "destination_url", Type: arrow.BinaryTypes.String},
        {Name: "destination_title", Type: arrow.BinaryTypes.String},
        {Name: "source_url", Type: arrow.BinaryTypes.String},
        {Name: "source_title", Type: arrow.BinaryTypes.String},
        {Name: "link_text", Type: arrow.BinaryTypes.String},
        {Name: "click_type", Type: arrow.BinaryTypes.String},
        {Name: "domain", Type: arrow.BinaryTypes.String},
        {Name: "is_new_tab", Type: arrow.FixedWidthTypes.Boolean},
        {Name: "timestamp", Type: arrow.FixedWidthTypes.Timestamp_ms},
        {Name: "created_at", Type: arrow.FixedWidthTypes.Timestamp_ms},
    }, nil)
}

func (bs *BookmarkService) writeBookmarksToParquet(bookmarks []Bookmark) error {
    schema := bs.getBookmarkSchema()
    mem := memory.NewGoAllocator()
    builder := array.NewRecordBuilder(mem, schema)
    defer builder.Release()

    for _, bookmark := range bookmarks {
        builder.Field(0).(*array.Int64Builder).Append(bookmark.ID)
        builder.Field(1).(*array.StringBuilder).Append(bookmark.URL)
        builder.Field(2).(*array.StringBuilder).Append(bookmark.Title)
        builder.Field(3).(*array.StringBuilder).Append(bookmark.Description)
        
        tagsJSON, _ := json.Marshal(bookmark.Tags)
        builder.Field(4).(*array.StringBuilder).Append(string(tagsJSON))
        
        builder.Field(5).(*array.TimestampBuilder).Append(arrow.Timestamp(bookmark.CreatedAt.UnixMilli()))
        builder.Field(6).(*array.TimestampBuilder).Append(arrow.Timestamp(bookmark.UpdatedAt.UnixMilli()))
    }

    record := builder.NewRecord()
    defer record.Release()

    filename := filepath.Join(bs.dataDir, "bookmarks.parquet")
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    writer, err := pqarrow.NewFileWriter(schema, file, parquet.NewWriterProperties(), pqarrow.DefaultWriterProps())
    if err != nil {
        return err
    }
    defer writer.Close()

    return writer.Write(record)
}

func (bs *BookmarkService) readBookmarksFromParquet() ([]Bookmark, error) {
    filename := filepath.Join(bs.dataDir, "bookmarks.parquet")
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        return []Bookmark{}, nil
    }

    fileReader, err := file.OpenParquetFile(filename, false)
    if err != nil {
        return nil, fmt.Errorf("failed to open parquet file: %w", err)
    }
    defer fileReader.Close()

    reader, err := pqarrow.NewFileReader(fileReader, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
    if err != nil {
        return nil, fmt.Errorf("failed to create parquet reader: %w", err)
    }

    table, err := reader.ReadTable(context.Background())
    if err != nil {
        return nil, fmt.Errorf("failed to read table: %w", err)
    }
    defer table.Release()

    var bookmarks []Bookmark
    
    if table.NumRows() == 0 {
        return bookmarks, nil
    }

    for i := 0; i < int(table.NumRows()); i++ {
        idCol := table.Column(0).Data().Chunk(0).(*array.Int64)
        urlCol := table.Column(1).Data().Chunk(0).(*array.String)
        titleCol := table.Column(2).Data().Chunk(0).(*array.String)
        descCol := table.Column(3).Data().Chunk(0).(*array.String)
        tagsCol := table.Column(4).Data().Chunk(0).(*array.String)
        createdCol := table.Column(5).Data().Chunk(0).(*array.Timestamp)
        updatedCol := table.Column(6).Data().Chunk(0).(*array.Timestamp)

        var tags []string
        if tagsJSON := tagsCol.Value(i); tagsJSON != "" {
            json.Unmarshal([]byte(tagsJSON), &tags)
        }

        bookmark := Bookmark{
            ID:          idCol.Value(i),
            URL:         urlCol.Value(i),
            Title:       titleCol.Value(i),
            Description: descCol.Value(i),
            Tags:        tags,
            CreatedAt:   time.UnixMilli(int64(createdCol.Value(i))),
            UpdatedAt:   time.UnixMilli(int64(updatedCol.Value(i))),
        }
        bookmarks = append(bookmarks, bookmark)
    }

    return bookmarks, nil
}

func (bs *BookmarkService) writeSessionsToParquet(sessions []Session) error {
    schema := bs.getSessionSchema()
    mem := memory.NewGoAllocator()
    builder := array.NewRecordBuilder(mem, schema)
    defer builder.Release()

    for _, session := range sessions {
        builder.Field(0).(*array.Int64Builder).Append(session.ID)
        builder.Field(1).(*array.StringBuilder).Append(session.Name)
        builder.Field(2).(*array.StringBuilder).Append(session.Description)
        
        tabsJSON, _ := json.Marshal(session.Tabs)
        builder.Field(3).(*array.StringBuilder).Append(string(tabsJSON))
        
        builder.Field(4).(*array.BooleanBuilder).Append(session.IsActive)
        builder.Field(5).(*array.TimestampBuilder).Append(arrow.Timestamp(session.CreatedAt.UnixMilli()))
        builder.Field(6).(*array.TimestampBuilder).Append(arrow.Timestamp(session.UpdatedAt.UnixMilli()))
    }

    record := builder.NewRecord()
    defer record.Release()

    filename := filepath.Join(bs.dataDir, "sessions.parquet")
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    writer, err := pqarrow.NewFileWriter(schema, file, parquet.NewWriterProperties(), pqarrow.DefaultWriterProps())
    if err != nil {
        return err
    }
    defer writer.Close()

    return writer.Write(record)
}

func (bs *BookmarkService) readSessionsFromParquet() ([]Session, error) {
    filename := filepath.Join(bs.dataDir, "sessions.parquet")
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        return []Session{}, nil
    }

    fileReader, err := file.OpenParquetFile(filename, false)
    if err != nil {
        return nil, fmt.Errorf("failed to open parquet file: %w", err)
    }
    defer fileReader.Close()

    reader, err := pqarrow.NewFileReader(fileReader, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
    if err != nil {
        return nil, fmt.Errorf("failed to create parquet reader: %w", err)
    }

    table, err := reader.ReadTable(context.Background())
    if err != nil {
        return nil, fmt.Errorf("failed to read table: %w", err)
    }
    defer table.Release()

    var sessions []Session
    
    if table.NumRows() == 0 {
        return sessions, nil
    }

    for i := 0; i < int(table.NumRows()); i++ {
        idCol := table.Column(0).Data().Chunk(0).(*array.Int64)
        nameCol := table.Column(1).Data().Chunk(0).(*array.String)
        descCol := table.Column(2).Data().Chunk(0).(*array.String)
        tabsCol := table.Column(3).Data().Chunk(0).(*array.String)
        activeCol := table.Column(4).Data().Chunk(0).(*array.Boolean)
        createdCol := table.Column(5).Data().Chunk(0).(*array.Timestamp)
        updatedCol := table.Column(6).Data().Chunk(0).(*array.Timestamp)

        var tabs []Tab
        if tabsJSON := tabsCol.Value(i); tabsJSON != "" {
            json.Unmarshal([]byte(tabsJSON), &tabs)
        }

        session := Session{
            ID:          idCol.Value(i),
            Name:        nameCol.Value(i),
            Description: descCol.Value(i),
            Tabs:        tabs,
            IsActive:    activeCol.Value(i),
            CreatedAt:   time.UnixMilli(int64(createdCol.Value(i))),
            UpdatedAt:   time.UnixMilli(int64(updatedCol.Value(i))),
        }
        sessions = append(sessions, session)
    }

    return sessions, nil
}

func (bs *BookmarkService) writeHistoryToParquet(history []HistoryEntry) error {
    schema := bs.getHistorySchema()
    mem := memory.NewGoAllocator()
    builder := array.NewRecordBuilder(mem, schema)
    defer builder.Release()

    for _, entry := range history {
        builder.Field(0).(*array.StringBuilder).Append(entry.URL)
        builder.Field(1).(*array.StringBuilder).Append(entry.Title)
        builder.Field(2).(*array.Int32Builder).Append(int32(entry.VisitCount))
        builder.Field(3).(*array.TimestampBuilder).Append(arrow.Timestamp(entry.LastVisitTime.UnixMilli()))
    }

    record := builder.NewRecord()
    defer record.Release()

    filename := filepath.Join(bs.dataDir, "history.parquet")
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    writer, err := pqarrow.NewFileWriter(schema, file, parquet.NewWriterProperties(), pqarrow.DefaultWriterProps())
    if err != nil {
        return err
    }
    defer writer.Close()

    return writer.Write(record)
}

func (bs *BookmarkService) readHistoryFromParquet() ([]HistoryEntry, error) {
    filename := filepath.Join(bs.dataDir, "history.parquet")
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        return []HistoryEntry{}, nil
    }

    fileReader, err := file.OpenParquetFile(filename, false)
    if err != nil {
        return nil, fmt.Errorf("failed to open parquet file: %w", err)
    }
    defer fileReader.Close()

    reader, err := pqarrow.NewFileReader(fileReader, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
    if err != nil {
        return nil, fmt.Errorf("failed to create parquet reader: %w", err)
    }

    table, err := reader.ReadTable(context.Background())
    if err != nil {
        return nil, fmt.Errorf("failed to read table: %w", err)
    }
    defer table.Release()

    var history []HistoryEntry
    
    if table.NumRows() == 0 {
        return history, nil
    }

    for i := 0; i < int(table.NumRows()); i++ {
        urlCol := table.Column(0).Data().Chunk(0).(*array.String)
        titleCol := table.Column(1).Data().Chunk(0).(*array.String)
        visitCountCol := table.Column(2).Data().Chunk(0).(*array.Int32)
        lastVisitCol := table.Column(3).Data().Chunk(0).(*array.Timestamp)

        entry := HistoryEntry{
            URL:           urlCol.Value(i),
            Title:         titleCol.Value(i),
            VisitCount:    int(visitCountCol.Value(i)),
            LastVisitTime: time.UnixMilli(int64(lastVisitCol.Value(i))),
        }
        history = append(history, entry)
    }

    return history, nil
}

func (bs *BookmarkService) enrichHistoryWithLinkClicks(history []HistoryEntry) ([]HistoryEntry, error) {
    // Get link clicks
    linkClicks, err := bs.readLinkClicksFromParquet()
    if err != nil {
        // If we can't read link clicks, just return history as-is
        return history, nil
    }
    
    // Create a map of destination URL to most recent link click
    clickMap := make(map[string]LinkClick)
    for _, click := range linkClicks {
        // Use the most recent click for each destination URL
        if existing, exists := clickMap[click.DestinationURL]; !exists || click.Timestamp.After(existing.Timestamp) {
            clickMap[click.DestinationURL] = click
        }
    }
    
    // Enrich history entries with link click data
    enrichedHistory := make([]HistoryEntry, len(history))
    for i, entry := range history {
        enrichedHistory[i] = entry
        if click, exists := clickMap[entry.URL]; exists {
            // Only add source info if the click happened on the same day
            // (Chrome's lastVisitTime updates on every visit, but we want the original click source)
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

// Helper function to get absolute duration
func abs(d time.Duration) time.Duration {
    if d < 0 {
        return -d
    }
    return d
}

func (bs *BookmarkService) writeLinkClicksToParquet(clicks []LinkClick) error {
    schema := bs.getLinkClickSchema()
    mem := memory.NewGoAllocator()
    builder := array.NewRecordBuilder(mem, schema)
    defer builder.Release()

    for _, click := range clicks {
        builder.Field(0).(*array.Int64Builder).Append(click.ID)
        builder.Field(1).(*array.StringBuilder).Append(click.DestinationURL)
        builder.Field(2).(*array.StringBuilder).Append(click.DestinationTitle)
        builder.Field(3).(*array.StringBuilder).Append(click.SourceURL)
        builder.Field(4).(*array.StringBuilder).Append(click.SourceTitle)
        builder.Field(5).(*array.StringBuilder).Append(click.LinkText)
        builder.Field(6).(*array.StringBuilder).Append(click.ClickType)
        builder.Field(7).(*array.StringBuilder).Append(click.Domain)
        builder.Field(8).(*array.BooleanBuilder).Append(click.IsNewTab)
        builder.Field(9).(*array.TimestampBuilder).Append(arrow.Timestamp(click.Timestamp.UnixMilli()))
        builder.Field(10).(*array.TimestampBuilder).Append(arrow.Timestamp(click.CreatedAt.UnixMilli()))
    }

    record := builder.NewRecord()
    defer record.Release()

    filename := filepath.Join(bs.dataDir, "link-clicks.parquet")
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    writer, err := pqarrow.NewFileWriter(schema, file, parquet.NewWriterProperties(), pqarrow.DefaultWriterProps())
    if err != nil {
        return err
    }
    defer writer.Close()

    return writer.Write(record)
}

func (bs *BookmarkService) readLinkClicksFromParquet() ([]LinkClick, error) {
    filename := filepath.Join(bs.dataDir, "link-clicks.parquet")
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        return []LinkClick{}, nil
    }

    fileReader, err := file.OpenParquetFile(filename, false)
    if err != nil {
        return nil, fmt.Errorf("failed to open parquet file: %w", err)
    }
    defer fileReader.Close()

    reader, err := pqarrow.NewFileReader(fileReader, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
    if err != nil {
        return nil, fmt.Errorf("failed to create parquet reader: %w", err)
    }

    table, err := reader.ReadTable(context.Background())
    if err != nil {
        return nil, fmt.Errorf("failed to read table: %w", err)
    }
    defer table.Release()

    var clicks []LinkClick
    
    if table.NumRows() == 0 {
        return clicks, nil
    }

    for i := 0; i < int(table.NumRows()); i++ {
        idCol := table.Column(0).Data().Chunk(0).(*array.Int64)
        destUrlCol := table.Column(1).Data().Chunk(0).(*array.String)
        destTitleCol := table.Column(2).Data().Chunk(0).(*array.String)
        srcUrlCol := table.Column(3).Data().Chunk(0).(*array.String)
        srcTitleCol := table.Column(4).Data().Chunk(0).(*array.String)
        linkTextCol := table.Column(5).Data().Chunk(0).(*array.String)
        clickTypeCol := table.Column(6).Data().Chunk(0).(*array.String)
        domainCol := table.Column(7).Data().Chunk(0).(*array.String)
        newTabCol := table.Column(8).Data().Chunk(0).(*array.Boolean)
        timestampCol := table.Column(9).Data().Chunk(0).(*array.Timestamp)
        createdCol := table.Column(10).Data().Chunk(0).(*array.Timestamp)

        click := LinkClick{
            ID:              idCol.Value(i),
            DestinationURL:  destUrlCol.Value(i),
            DestinationTitle: destTitleCol.Value(i),
            SourceURL:       srcUrlCol.Value(i),
            SourceTitle:     srcTitleCol.Value(i),
            LinkText:        linkTextCol.Value(i),
            ClickType:       clickTypeCol.Value(i),
            Domain:          domainCol.Value(i),
            IsNewTab:        newTabCol.Value(i),
            Timestamp:       time.UnixMilli(int64(timestampCol.Value(i))),
            CreatedAt:       time.UnixMilli(int64(createdCol.Value(i))),
        }
        clicks = append(clicks, click)
    }

    return clicks, nil
}

func (bs *BookmarkService) getAllBookmarks(w http.ResponseWriter, r *http.Request) {
    bookmarks, err := bs.readBookmarksFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(bookmarks)
}

func (bs *BookmarkService) createBookmark(w http.ResponseWriter, r *http.Request) {
    var bookmark Bookmark
    if err := json.NewDecoder(r.Body).Decode(&bookmark); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    bookmark.ID = time.Now().UnixNano()
    bookmark.CreatedAt = time.Now()
    bookmark.UpdatedAt = time.Now()

    bookmarks, err := bs.readBookmarksFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    bookmarks = append(bookmarks, bookmark)

    if err := bs.writeBookmarksToParquet(bookmarks); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(bookmark)
}

func (bs *BookmarkService) updateBookmark(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
        return
    }

    var updatedBookmark Bookmark
    if err := json.NewDecoder(r.Body).Decode(&updatedBookmark); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    bookmarks, err := bs.readBookmarksFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    for i, bookmark := range bookmarks {
        if bookmark.ID == id {
            updatedBookmark.ID = id
            updatedBookmark.CreatedAt = bookmark.CreatedAt
            updatedBookmark.UpdatedAt = time.Now()
            bookmarks[i] = updatedBookmark
            
            if err := bs.writeBookmarksToParquet(bookmarks); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }

            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(updatedBookmark)
            return
        }
    }

    http.Error(w, "Bookmark not found", http.StatusNotFound)
}

func (bs *BookmarkService) deleteBookmark(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
        return
    }

    bookmarks, err := bs.readBookmarksFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    for i, bookmark := range bookmarks {
        if bookmark.ID == id {
            bookmarks = append(bookmarks[:i], bookmarks[i+1:]...)
            
            if err := bs.writeBookmarksToParquet(bookmarks); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }

            w.WriteHeader(http.StatusNoContent)
            return
        }
    }

    http.Error(w, "Bookmark not found", http.StatusNotFound)
}

func (bs *BookmarkService) searchBookmarks(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    if query == "" {
        http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
        return
    }

    bookmarks, err := bs.readBookmarksFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    var results []Bookmark
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

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(results)
}

func (bs *BookmarkService) saveSession(w http.ResponseWriter, r *http.Request) {
    var session Session
    if err := json.NewDecoder(r.Body).Decode(&session); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    session.ID = time.Now().UnixNano()
    session.CreatedAt = time.Now()
    session.UpdatedAt = time.Now()
    session.IsActive = true

    sessions, err := bs.readSessionsFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    sessions = append(sessions, session)

    if err := bs.writeSessionsToParquet(sessions); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(session)
}

func (bs *BookmarkService) getAllSessions(w http.ResponseWriter, r *http.Request) {
    sessions, err := bs.readSessionsFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(sessions)
}

func (bs *BookmarkService) updateSession(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, "Invalid session ID", http.StatusBadRequest)
        return
    }

    var updatedSession Session
    if err := json.NewDecoder(r.Body).Decode(&updatedSession); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    sessions, err := bs.readSessionsFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    for i, session := range sessions {
        if session.ID == id {
            updatedSession.ID = id
            updatedSession.CreatedAt = session.CreatedAt
            updatedSession.UpdatedAt = time.Now()
            sessions[i] = updatedSession
            
            if err := bs.writeSessionsToParquet(sessions); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }

            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(updatedSession)
            return
        }
    }

    http.Error(w, "Session not found", http.StatusNotFound)
}

func (bs *BookmarkService) deleteSession(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, "Invalid session ID", http.StatusBadRequest)
        return
    }

    sessions, err := bs.readSessionsFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    for i, session := range sessions {
        if session.ID == id {
            sessions = append(sessions[:i], sessions[i+1:]...)
            
            if err := bs.writeSessionsToParquet(sessions); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }

            w.WriteHeader(http.StatusNoContent)
            return
        }
    }

    http.Error(w, "Session not found", http.StatusNotFound)
}

func (bs *BookmarkService) getAllHistory(w http.ResponseWriter, r *http.Request) {
    history, err := bs.readHistoryFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Enrich with link click data
    enrichedHistory, err := bs.enrichHistoryWithLinkClicks(history)
    if err != nil {
        enrichedHistory = history
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(enrichedHistory)
}

func (bs *BookmarkService) getTodaysHistory(w http.ResponseWriter, r *http.Request) {
    history, err := bs.readHistoryFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    today := time.Now().Truncate(24 * time.Hour)
    tomorrow := today.Add(24 * time.Hour)

    var todaysHistory []HistoryEntry
    for _, entry := range history {
        if entry.LastVisitTime.After(today) && entry.LastVisitTime.Before(tomorrow) {
            todaysHistory = append(todaysHistory, entry)
        }
    }

    // Enrich with link click data
    enrichedHistory, err := bs.enrichHistoryWithLinkClicks(todaysHistory)
    if err != nil {
        // If enrichment fails, just return the original history
        enrichedHistory = todaysHistory
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(enrichedHistory)
}

func (bs *BookmarkService) getWeekHistory(w http.ResponseWriter, r *http.Request) {
    history, err := bs.readHistoryFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    now := time.Now()
    weekAgo := now.AddDate(0, 0, -7)

    var weekHistory []HistoryEntry
    for _, entry := range history {
        if entry.LastVisitTime.After(weekAgo) {
            weekHistory = append(weekHistory, entry)
        }
    }

    // Enrich with link click data
    enrichedHistory, err := bs.enrichHistoryWithLinkClicks(weekHistory)
    if err != nil {
        enrichedHistory = weekHistory
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(enrichedHistory)
}

func (bs *BookmarkService) getMonthHistory(w http.ResponseWriter, r *http.Request) {
    history, err := bs.readHistoryFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    now := time.Now()
    monthAgo := now.AddDate(0, 0, -30)

    var monthHistory []HistoryEntry
    for _, entry := range history {
        if entry.LastVisitTime.After(monthAgo) {
            monthHistory = append(monthHistory, entry)
        }
    }

    // Enrich with link click data
    enrichedHistory, err := bs.enrichHistoryWithLinkClicks(monthHistory)
    if err != nil {
        enrichedHistory = monthHistory
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(enrichedHistory)
}

func (bs *BookmarkService) getHistoryCount(w http.ResponseWriter, r *http.Request) {
    history, err := bs.readHistoryFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    response := map[string]interface{}{
        "total_count": len(history),
        "message":     "History count retrieved successfully",
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (bs *BookmarkService) syncHistory(w http.ResponseWriter, r *http.Request) {
    var historyRequest struct {
        History []HistoryEntry `json:"history"`
    }

    if err := json.NewDecoder(r.Body).Decode(&historyRequest); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    existingHistory, err := bs.readHistoryFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    historyMap := make(map[string]HistoryEntry)
    for _, entry := range existingHistory {
        historyMap[entry.URL] = entry
    }

    for _, newEntry := range historyRequest.History {
        if existing, exists := historyMap[newEntry.URL]; exists {
            if newEntry.LastVisitTime.After(existing.LastVisitTime) {
                historyMap[newEntry.URL] = newEntry
            }
        } else {
            historyMap[newEntry.URL] = newEntry
        }
    }

    var allHistory []HistoryEntry
    for _, entry := range historyMap {
        allHistory = append(allHistory, entry)
    }

    if err := bs.writeHistoryToParquet(allHistory); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    response := map[string]interface{}{
        "synced_count": len(historyRequest.History),
        "total_count":  len(allHistory),
        "message":      "History synchronized successfully",
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (bs *BookmarkService) syncLinkClicks(w http.ResponseWriter, r *http.Request) {
    var clickRequest struct {
        Clicks []struct {
            DestinationURL   string `json:"destinationUrl"`
            DestinationTitle string `json:"destinationTitle"`
            SourceURL        string `json:"sourceUrl"`
            SourceTitle      string `json:"sourceTitle"`
            LinkText         string `json:"linkText"`
            ClickType        string `json:"clickType"`
            Domain           string `json:"domain"`
            IsNewTab         bool   `json:"isNewTab"`
            Timestamp        int64  `json:"timestamp"`
        } `json:"clicks"`
    }

    if err := json.NewDecoder(r.Body).Decode(&clickRequest); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    existingClicks, err := bs.readLinkClicksFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Convert incoming clicks to LinkClick structs
    var newClicks []LinkClick
    for _, click := range clickRequest.Clicks {
        newClick := LinkClick{
            ID:               time.Now().UnixNano(),
            DestinationURL:   click.DestinationURL,
            DestinationTitle: click.DestinationTitle,
            SourceURL:        click.SourceURL,
            SourceTitle:      click.SourceTitle,
            LinkText:         click.LinkText,
            ClickType:        click.ClickType,
            Domain:           click.Domain,
            IsNewTab:         click.IsNewTab,
            Timestamp:        time.UnixMilli(click.Timestamp),
            CreatedAt:        time.Now(),
        }
        newClicks = append(newClicks, newClick)
    }

    allClicks := append(existingClicks, newClicks...)
    
    if err := bs.writeLinkClicksToParquet(allClicks); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    response := map[string]interface{}{
        "synced_count": len(newClicks),
        "total_count":  len(allClicks),
        "message":      "Link clicks synchronized successfully",
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (bs *BookmarkService) getAllLinkClicks(w http.ResponseWriter, r *http.Request) {
    clicks, err := bs.readLinkClicksFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(clicks)
}

func (bs *BookmarkService) importBrowserData(w http.ResponseWriter, r *http.Request) {
    var importRequest struct {
        Bookmarks []ImportedBookmark `json:"bookmarks"`
        History   []HistoryEntry     `json:"history"`
        UseAI     bool               `json:"use_ai"`
    }

    if err := json.NewDecoder(r.Body).Decode(&importRequest); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    existingBookmarks, err := bs.readBookmarksFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    var newBookmarks []Bookmark
    for _, imported := range importRequest.Bookmarks {
        bookmark := Bookmark{
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
    
    if err := bs.writeBookmarksToParquet(allBookmarks); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    response := map[string]interface{}{
        "imported_count": len(newBookmarks),
        "total_count":    len(allBookmarks),
        "ai_processed":   importRequest.UseAI,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (bs *BookmarkService) bulkSegmentBookmarks(w http.ResponseWriter, r *http.Request) {
    bookmarks, err := bs.readBookmarksFromParquet()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    processedCount := 0
    for i, bookmark := range bookmarks {
        if len(bookmark.Tags) == 0 {
            bookmarks[i].Tags = bs.generateTagsFromContent(bookmark)
            bookmarks[i].UpdatedAt = time.Now()
            processedCount++
        }
    }

    if processedCount > 0 {
        if err := bs.writeBookmarksToParquet(bookmarks); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    }

    response := map[string]interface{}{
        "processed_count": processedCount,
        "total_count":     len(bookmarks),
        "message":         "AI segmentation completed",
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (bs *BookmarkService) generateTagsFromContent(bookmark Bookmark) []string {
    tags := []string{}
    
    content := strings.ToLower(bookmark.Title + " " + bookmark.Description + " " + bookmark.URL)
    
    keywords := map[string]string{
        "github.com":     "development",
        "stackoverflow":  "development",
        "youtube.com":    "video",
        "medium.com":     "article",
        "news":           "news",
        "blog":           "blog",
        "tutorial":       "learning",
        "documentation":  "docs",
        "api":            "development",
        "react":          "frontend",
        "javascript":     "development",
        "python":         "development",
        "golang":         "development",
        "design":         "design",
        "tool":           "tools",
    }
    
    for keyword, tag := range keywords {
        if strings.Contains(content, keyword) {
            tags = append(tags, tag)
        }
    }
    
    if len(tags) == 0 {
        tags = append(tags, "uncategorized")
    }
    
    return tags
}

func main() {
    dataDir := os.Getenv("DATA_DIR")
    if dataDir == "" {
        dataDir = "./data"
    }
    service := NewBookmarkService(dataDir)

    router := mux.NewRouter()

    router.HandleFunc("/api/bookmarks", service.getAllBookmarks).Methods("GET")
    router.HandleFunc("/api/bookmarks", service.createBookmark).Methods("POST")
    router.HandleFunc("/api/bookmarks/{id}", service.updateBookmark).Methods("PUT")
    router.HandleFunc("/api/bookmarks/{id}", service.deleteBookmark).Methods("DELETE")
    router.HandleFunc("/api/bookmarks/search", service.searchBookmarks).Methods("GET")

    router.HandleFunc("/api/sessions", service.getAllSessions).Methods("GET")
    router.HandleFunc("/api/sessions", service.saveSession).Methods("POST")
    router.HandleFunc("/api/sessions/{id}", service.updateSession).Methods("PUT")
    router.HandleFunc("/api/sessions/{id}", service.deleteSession).Methods("DELETE")
    
    router.HandleFunc("/api/history", service.getAllHistory).Methods("GET")
    router.HandleFunc("/api/history/today", service.getTodaysHistory).Methods("GET")
    router.HandleFunc("/api/history/week", service.getWeekHistory).Methods("GET")
    router.HandleFunc("/api/history/month", service.getMonthHistory).Methods("GET")
    router.HandleFunc("/api/history/count", service.getHistoryCount).Methods("GET")
    router.HandleFunc("/api/history/sync", service.syncHistory).Methods("POST")
    
    router.HandleFunc("/api/link-clicks", service.getAllLinkClicks).Methods("GET")
    router.HandleFunc("/api/link-clicks/sync", service.syncLinkClicks).Methods("POST")
    
    router.HandleFunc("/api/import/browser", service.importBrowserData).Methods("POST")
    router.HandleFunc("/api/ai/segment", service.bulkSegmentBookmarks).Methods("POST")
    
    // Health/ping endpoint for extension testing
    router.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{"status": "ok", "timestamp": time.Now().UTC().Format(time.RFC3339)})
    }).Methods("GET")
    
    router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
    }).Methods("GET")

    c := cors.New(cors.Options{
        AllowedOrigins: []string{"*"},
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders: []string{"*"},
        AllowCredentials: false,
        MaxAge: 86400,
    })

    handler := c.Handler(router)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    fmt.Printf("🚀 HyprLnk API starting on port %s\n", port)
    log.Fatal(http.ListenAndServe(":"+port, handler))
}