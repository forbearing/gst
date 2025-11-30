package modelaichat

import "github.com/forbearing/gst/model"

// DocumentStatus represents the processing status of a document
type DocumentStatus string

const (
	DocumentStatusPending   DocumentStatus = "pending"   // Pending processing
	DocumentStatusParsing   DocumentStatus = "parsing"   // Parsing
	DocumentStatusChunking  DocumentStatus = "chunking"  // Chunking
	DocumentStatusEmbedding DocumentStatus = "embedding" // Embedding
	DocumentStatusIndexing  DocumentStatus = "indexing"  // Indexing
	DocumentStatusCompleted DocumentStatus = "completed" // Completed
	DocumentStatusFailed    DocumentStatus = "failed"    // Failed
)

// DocumentType represents the type of document
type DocumentType string

const (
	DocumentTypePDF      DocumentType = "pdf"      // PDF document
	DocumentTypeWord     DocumentType = "word"     // Word document
	DocumentTypeMarkdown DocumentType = "markdown" // Markdown document
	DocumentTypeText     DocumentType = "text"     // Text document
	DocumentTypeHTML     DocumentType = "html"     // HTML document
	DocumentTypeExcel    DocumentType = "excel"    // Excel document
	DocumentTypeCSV      DocumentType = "csv"      // CSV document
)

// Document represents a document in a knowledge base
type Document struct {
	KnowledgeBaseID string         `gorm:"size:100;not null;index" json:"knowledge_base_id" schema:"knowledge_base_id"` // Knowledge base ID
	Name            string         `gorm:"size:255;not null" json:"name" schema:"name"`                                 // Document name
	Type            DocumentType   `gorm:"size:20;not null" json:"type" schema:"type"`                                  // Document type
	Status          DocumentStatus `gorm:"size:20;default:pending" json:"status,omitempty" schema:"status"`             // Processing status
	ErrMessage      string         `gorm:"size:500" json:"err_message,omitempty"`                                       // Error message if failed

	// File information
	FilePath     string `gorm:"size:500" json:"file_path,omitempty"`      // File path
	FileSize     int64  `gorm:"default:0" json:"file_size,omitempty"`     // File size in bytes
	FileHash     string `gorm:"size:64;index" json:"file_hash,omitempty"` // File hash (SHA256)
	ContentType  string `gorm:"size:100" json:"content_type,omitempty"`   // Content type (MIME type)
	OriginalName string `gorm:"size:255" json:"original_name,omitempty"`  // Original file name

	// Processing results
	ChunkCount int `gorm:"default:0" json:"chunk_count,omitempty"` // Chunk count
	CharCount  int `gorm:"default:0" json:"char_count,omitempty"`  // Character count
	TokenCount int `gorm:"default:0" json:"token_count,omitempty"` // Token count
	PageCount  int `gorm:"default:0" json:"page_count,omitempty"`  // Page count

	// Processing progress
	Progress      int   `gorm:"default:0" json:"progress,omitempty"` // Progress percentage (0-100)
	ProcessedAt   int64 `json:"processed_at,omitempty"`              // Processed timestamp
	ProcessTimeMs int64 `json:"process_time_ms,omitempty"`           // Processing time in milliseconds

	KnowledgeBase *KnowledgeBase `gorm:"-" json:"knowledge_base,omitempty"` // Associated knowledge base
	Chunks        []Chunk        `gorm:"-" json:"chunks,omitempty"`         // Associated chunks

	model.Base
}

func (Document) Purge() bool          { return true }
func (Document) GetTableName() string { return "ai_rag_documents" }
