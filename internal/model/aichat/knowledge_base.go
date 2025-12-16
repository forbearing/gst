package modelaichat

import "github.com/forbearing/gst/model"

// KnowledgeBaseType represents the type of knowledge base
type KnowledgeBaseType string

const (
	KnowledgeBaseTypePersonal KnowledgeBaseType = "personal" // Personal knowledge base
	KnowledgeBaseTypeTeam     KnowledgeBaseType = "team"     // Team knowledge base
	KnowledgeBaseTypePublic   KnowledgeBaseType = "public"   // Public knowledge base
)

// EmbeddingModel represents the embedding model type
type EmbeddingModel string

const (
	EmbeddingModelOpenAI EmbeddingModel = "text-embedding-3-small" // OpenAI embedding model
	EmbeddingModelBGE    EmbeddingModel = "bge-large-zh"           // BGE large Chinese embedding model
)

// KnowledgeBase represents a knowledge base for RAG (Retrieval-Augmented Generation)
type KnowledgeBase struct {
	Name        string            `gorm:"size:100;not null" json:"name" schema:"name"`                       // Knowledge base name
	Description string            `gorm:"size:500" json:"description,omitempty" schema:"description"`        // Description
	UserID      string            `gorm:"size:100;not null;index" json:"user_id,omitempty" schema:"user_id"` // User ID
	Type        KnowledgeBaseType `gorm:"size:20;default:personal" json:"type,omitempty" schema:"type"`      // Knowledge base type

	// Embedding configuration
	EmbeddingModel EmbeddingModel `gorm:"size:50;default:text-embedding-3-small" json:"embedding_model,omitempty" schema:"embedding_model"` // Embedding model
	VectorDim      int            `gorm:"default:1536" json:"vector_dim,omitempty" schema:"vector_dim"`                                     // Vector dimension
	ChunkSize      int            `gorm:"default:500" json:"chunk_size,omitempty" schema:"chunk_size"`                                      // Chunk size
	ChunkOverlap   int            `gorm:"default:50" json:"chunk_overlap,omitempty" schema:"chunk_overlap"`                                 // Chunk overlap

	// Statistics
	DocumentCount int   `gorm:"default:0" json:"document_count,omitempty"` // Document count
	ChunkCount    int   `gorm:"default:0" json:"chunk_count,omitempty"`    // Chunk count
	TotalSize     int64 `gorm:"default:0" json:"total_size,omitempty"`     // Total size in bytes

	Documents []Document `gorm:"-" json:"documents,omitempty"` // Associated documents

	model.Base
}

func (KnowledgeBase) Purge() bool          { return true }
func (KnowledgeBase) GetTableName() string { return "ai_rag_knowledge_bases" }
