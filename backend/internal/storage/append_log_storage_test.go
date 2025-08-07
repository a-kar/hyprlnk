package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hyprlnk/internal/models"
)

func TestAppendLogStorage_BookmarkOperations(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "hyprlink_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create storage
	storage := NewAppendLogStorage(tempDir)
	defer storage.Close()

	// Test 1: Add bookmark
	bookmark1 := models.Bookmark{
		URL:         "https://example.com",
		Title:       "Example Site",
		Description: "Test bookmark",
		Tags:        []string{"test", "example"},
	}

	err = storage.AddBookmark(bookmark1)
	if err != nil {
		t.Fatalf("Failed to add bookmark: %v", err)
	}

	// Test 2: Read bookmarks (should see the added bookmark)
	bookmarks, err := storage.ReadBookmarks()
	if err != nil {
		t.Fatalf("Failed to read bookmarks: %v", err)
	}

	if len(bookmarks) != 1 {
		t.Fatalf("Expected 1 bookmark, got %d", len(bookmarks))
	}

	if bookmarks[0].URL != "https://example.com" {
		t.Fatalf("Expected URL 'https://example.com', got '%s'", bookmarks[0].URL)
	}

	savedID := bookmarks[0].ID

	// Test 3: Update bookmark
	bookmarks[0].Title = "Updated Example"
	err = storage.UpdateBookmark(bookmarks[0])
	if err != nil {
		t.Fatalf("Failed to update bookmark: %v", err)
	}

	// Read again to verify update
	bookmarks, err = storage.ReadBookmarks()
	if err != nil {
		t.Fatalf("Failed to read bookmarks after update: %v", err)
	}

	if len(bookmarks) != 1 {
		t.Fatalf("Expected 1 bookmark after update, got %d", len(bookmarks))
	}

	if bookmarks[0].Title != "Updated Example" {
		t.Fatalf("Expected title 'Updated Example', got '%s'", bookmarks[0].Title)
	}

	// Test 4: Add another bookmark
	bookmark2 := models.Bookmark{
		URL:   "https://github.com",
		Title: "GitHub",
		Tags:  []string{"dev"},
	}

	err = storage.AddBookmark(bookmark2)
	if err != nil {
		t.Fatalf("Failed to add second bookmark: %v", err)
	}

	bookmarks, err = storage.ReadBookmarks()
	if err != nil {
		t.Fatalf("Failed to read bookmarks: %v", err)
	}

	if len(bookmarks) != 2 {
		t.Fatalf("Expected 2 bookmarks, got %d", len(bookmarks))
	}

	// Test 5: Delete bookmark
	err = storage.DeleteBookmark(savedID)
	if err != nil {
		t.Fatalf("Failed to delete bookmark: %v", err)
	}

	bookmarks, err = storage.ReadBookmarks()
	if err != nil {
		t.Fatalf("Failed to read bookmarks after delete: %v", err)
	}

	if len(bookmarks) != 1 {
		t.Fatalf("Expected 1 bookmark after delete, got %d", len(bookmarks))
	}

	if bookmarks[0].URL != "https://github.com" {
		t.Fatalf("Wrong bookmark remained after delete, expected GitHub, got %s", bookmarks[0].URL)
	}

	// Test 6: Verify delta file exists
	deltaFile := filepath.Join(tempDir, "bookmarks.delta.json")
	if _, err := os.Stat(deltaFile); os.IsNotExist(err) {
		t.Fatal("Delta file should exist")
	}

	// Test 7: Simulate restart by creating new storage instance
	storage.Close()
	
	storage2 := NewAppendLogStorage(tempDir)
	defer storage2.Close()

	// Should still see the same bookmarks after restart
	bookmarks, err = storage2.ReadBookmarks()
	if err != nil {
		t.Fatalf("Failed to read bookmarks after restart: %v", err)
	}

	if len(bookmarks) != 1 {
		t.Fatalf("Expected 1 bookmark after restart, got %d", len(bookmarks))
	}

	if bookmarks[0].URL != "https://github.com" {
		t.Fatalf("Expected GitHub bookmark after restart, got %s", bookmarks[0].URL)
	}
}

func TestAppendLogStorage_Compaction(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "hyprlink_compact_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create storage with low compaction threshold for testing
	storage := NewAppendLogStorage(tempDir)
	storage.compactThreshold = 5 // Compact after 5 operations
	defer storage.Close()

	// Add several bookmarks to trigger compaction
	for i := 0; i < 10; i++ {
		bookmark := models.Bookmark{
			URL:   fmt.Sprintf("https://example%d.com", i),
			Title: fmt.Sprintf("Example %d", i),
		}
		err := storage.AddBookmark(bookmark)
		if err != nil {
			t.Fatalf("Failed to add bookmark %d: %v", i, err)
		}
	}

	// Wait a bit for background compaction to complete
	time.Sleep(100 * time.Millisecond)

	// Check that main Parquet file exists after compaction
	mainFile := filepath.Join(tempDir, "bookmarks.parquet")
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		t.Fatal("Main Parquet file should exist after compaction")
	}

	// Verify all bookmarks are still accessible
	bookmarks, err := storage.ReadBookmarks()
	if err != nil {
		t.Fatalf("Failed to read bookmarks after compaction: %v", err)
	}

	if len(bookmarks) != 10 {
		t.Fatalf("Expected 10 bookmarks after compaction, got %d", len(bookmarks))
	}
}

func BenchmarkAppendLogStorage_SingleWrites(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "hyprlink_bench")
	defer os.RemoveAll(tempDir)

	storage := NewAppendLogStorage(tempDir)
	defer storage.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bookmark := models.Bookmark{
			URL:   fmt.Sprintf("https://example%d.com", i),
			Title: fmt.Sprintf("Benchmark %d", i),
		}
		storage.AddBookmark(bookmark)
	}
}

func BenchmarkParquetStorage_SingleWrites(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "hyprlink_bench_parquet")
	defer os.RemoveAll(tempDir)

	storage := NewParquetStorage(tempDir)

	// Pre-populate with some data for realistic comparison
	initial := make([]models.Bookmark, 100)
	for i := range initial {
		initial[i] = models.Bookmark{
			ID:    int64(i),
			URL:   fmt.Sprintf("https://initial%d.com", i),
			Title: fmt.Sprintf("Initial %d", i),
		}
	}
	storage.WriteBookmarks(initial)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Read all, add one, write all (old approach)
		bookmarks, _ := storage.ReadBookmarks()
		bookmark := models.Bookmark{
			ID:    int64(time.Now().UnixNano()),
			URL:   fmt.Sprintf("https://example%d.com", i),
			Title: fmt.Sprintf("Benchmark %d", i),
		}
		bookmarks = append(bookmarks, bookmark)
		storage.WriteBookmarks(bookmarks)
	}
}