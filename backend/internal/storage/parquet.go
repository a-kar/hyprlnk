package storage

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/apache/arrow/go/v14/arrow"
    "github.com/apache/arrow/go/v14/arrow/array"
    "github.com/apache/arrow/go/v14/arrow/memory"
    "github.com/apache/arrow/go/v14/parquet"
    "github.com/apache/arrow/go/v14/parquet/file"
    "github.com/apache/arrow/go/v14/parquet/pqarrow"

    "hyprlnk/internal/models"
)

type ParquetStorage struct {
    dataDir string
}

func NewParquetStorage(dataDir string) *ParquetStorage {
    os.MkdirAll(dataDir, 0755)
    return &ParquetStorage{dataDir: dataDir}
}

func (ps *ParquetStorage) getBookmarkSchema() *arrow.Schema {
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

func (ps *ParquetStorage) getSessionSchema() *arrow.Schema {
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

func (ps *ParquetStorage) getHistorySchema() *arrow.Schema {
    return arrow.NewSchema([]arrow.Field{
        {Name: "url", Type: arrow.BinaryTypes.String},
        {Name: "title", Type: arrow.BinaryTypes.String},
        {Name: "visit_count", Type: arrow.PrimitiveTypes.Int32},
        {Name: "last_visit_time", Type: arrow.FixedWidthTypes.Timestamp_ms},
    }, nil)
}

func (ps *ParquetStorage) getLinkClickSchema() *arrow.Schema {
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

func (ps *ParquetStorage) WriteBookmarks(bookmarks []models.Bookmark) error {
    schema := ps.getBookmarkSchema()
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

    filename := filepath.Join(ps.dataDir, "bookmarks.parquet")
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

func (ps *ParquetStorage) ReadBookmarks() ([]models.Bookmark, error) {
    filename := filepath.Join(ps.dataDir, "bookmarks.parquet")
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        return []models.Bookmark{}, nil
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

    var bookmarks []models.Bookmark
    
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

        bookmark := models.Bookmark{
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

func (ps *ParquetStorage) WriteSessions(sessions []models.Session) error {
    schema := ps.getSessionSchema()
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

    filename := filepath.Join(ps.dataDir, "sessions.parquet")
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

func (ps *ParquetStorage) ReadSessions() ([]models.Session, error) {
    filename := filepath.Join(ps.dataDir, "sessions.parquet")
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        return []models.Session{}, nil
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

    var sessions []models.Session
    
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

        var tabs []models.Tab
        if tabsJSON := tabsCol.Value(i); tabsJSON != "" {
            json.Unmarshal([]byte(tabsJSON), &tabs)
        }

        session := models.Session{
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

func (ps *ParquetStorage) WriteHistory(history []models.HistoryEntry) error {
    schema := ps.getHistorySchema()
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

    filename := filepath.Join(ps.dataDir, "history.parquet")
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

func (ps *ParquetStorage) ReadHistory() ([]models.HistoryEntry, error) {
    filename := filepath.Join(ps.dataDir, "history.parquet")
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        return []models.HistoryEntry{}, nil
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

    var history []models.HistoryEntry
    
    if table.NumRows() == 0 {
        return history, nil
    }

    for i := 0; i < int(table.NumRows()); i++ {
        urlCol := table.Column(0).Data().Chunk(0).(*array.String)
        titleCol := table.Column(1).Data().Chunk(0).(*array.String)
        visitCountCol := table.Column(2).Data().Chunk(0).(*array.Int32)
        lastVisitCol := table.Column(3).Data().Chunk(0).(*array.Timestamp)

        entry := models.HistoryEntry{
            URL:           urlCol.Value(i),
            Title:         titleCol.Value(i),
            VisitCount:    int(visitCountCol.Value(i)),
            LastVisitTime: time.UnixMilli(int64(lastVisitCol.Value(i))),
        }
        history = append(history, entry)
    }

    return history, nil
}

func (ps *ParquetStorage) WriteLinkClicks(clicks []models.LinkClick) error {
    schema := ps.getLinkClickSchema()
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

    filename := filepath.Join(ps.dataDir, "link-clicks.parquet")
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

func (ps *ParquetStorage) ReadLinkClicks() ([]models.LinkClick, error) {
    filename := filepath.Join(ps.dataDir, "link-clicks.parquet")
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        return []models.LinkClick{}, nil
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

    var clicks []models.LinkClick
    
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

        click := models.LinkClick{
            ID:               idCol.Value(i),
            DestinationURL:   destUrlCol.Value(i),
            DestinationTitle: destTitleCol.Value(i),
            SourceURL:        srcUrlCol.Value(i),
            SourceTitle:      srcTitleCol.Value(i),
            LinkText:         linkTextCol.Value(i),
            ClickType:        clickTypeCol.Value(i),
            Domain:           domainCol.Value(i),
            IsNewTab:         newTabCol.Value(i),
            Timestamp:        time.UnixMilli(int64(timestampCol.Value(i))),
            CreatedAt:        time.UnixMilli(int64(createdCol.Value(i))),
        }
        clicks = append(clicks, click)
    }

    return clicks, nil
}