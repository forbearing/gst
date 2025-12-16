package modelaichat

import (
	"github.com/forbearing/gst/model"
)

// Chunk represents a text chunk from a document for RAG (Retrieval-Augmented Generation)
type Chunk struct {
	KnowledgeBaseID string `gorm:"size:100;not null;index" json:"knowledge_base_id" schema:"knowledge_base_id"` // Knowledge base ID
	DocumentID      string `gorm:"size:100;not null;index" json:"document_id" schema:"document_id"`             // Document ID
	Content         string `gorm:"type:text;not null" json:"content" schema:"content"`                          // Chunk content
	// Embedding       pgvector.Vector `gorm:"type:vector(1536)" json:"-"` // Vector embedding (commented out, requires pgvector extension)

	// Position information
	ChunkIndex int `gorm:"not null" json:"chunk_index" schema:"chunk_index"` // Chunk index in document
	StartPos   int `gorm:"default:0" json:"start_pos,omitempty"`             // Start position in original document
	EndPos     int `gorm:"default:0" json:"end_pos,omitempty"`               // End position in original document
	PageNum    int `gorm:"default:0" json:"page_num,omitempty"`              // Page number (for PDF documents)

	// Statistics
	CharCount  int `gorm:"default:0" json:"char_count,omitempty"`  // Character count
	TokenCount int `gorm:"default:0" json:"token_count,omitempty"` // Token count

	KnowledgeBase *KnowledgeBase `gorm:"-" json:"knowledge_base,omitempty"` // Associated knowledge base
	Document      *Document      `gorm:"-" json:"document,omitempty"`       // Associated document

	model.Base
}

func (Chunk) Purge() bool          { return true }
func (Chunk) GetTableName() string { return "ai_rag_chunks" }
