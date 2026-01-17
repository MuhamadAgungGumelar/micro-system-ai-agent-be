package vector

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
)

// Service provides high-level vector database operations
type Service struct {
	provider  Provider
	embedding EmbeddingProvider
}

// NewService creates a new vector service
func NewService(provider Provider, embedding EmbeddingProvider) *Service {
	return &Service{
		provider:  provider,
		embedding: embedding,
	}
}

// Initialize initializes both provider and embedding service
func (s *Service) Initialize(ctx context.Context) error {
	log.Printf("📊 Initializing Vector DB (%s)...", s.provider.GetProviderType())

	if err := s.provider.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize vector provider: %w", err)
	}

	log.Printf("✅ Vector DB initialized successfully")
	log.Printf("📐 Embedding model: %s (%d dimensions)", s.embedding.GetProviderName(), s.embedding.GetDimensions())

	return nil
}

// CreateCollection creates a new collection with the embedding dimensions
func (s *Service) CreateCollection(ctx context.Context, name string) error {
	return s.provider.CreateCollection(ctx, name, s.embedding.GetDimensions())
}

// DeleteCollection deletes a collection
func (s *Service) DeleteCollection(ctx context.Context, name string) error {
	return s.provider.DeleteCollection(ctx, name)
}

// AddDocument adds a document to the collection
func (s *Service) AddDocument(ctx context.Context, collection, documentID, text string, metadata map[string]interface{}) error {
	// Generate embedding
	embedding, err := s.embedding.GenerateEmbedding(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Prepare payload (metadata + original text)
	payload := make(map[string]interface{})
	for k, v := range metadata {
		payload[k] = v
	}
	payload["text"] = text

	// Upsert point
	points := []Point{
		{
			ID:      documentID,
			Vector:  embedding,
			Payload: payload,
		},
	}

	return s.provider.Upsert(ctx, collection, points)
}

// AddDocuments adds multiple documents in batch
func (s *Service) AddDocuments(ctx context.Context, collection string, documents []Document) error {
	if len(documents) == 0 {
		return fmt.Errorf("no documents to add")
	}

	// Extract texts for batch embedding generation
	texts := make([]string, len(documents))
	for i, doc := range documents {
		texts[i] = doc.Text
	}

	// Generate embeddings in batch
	embeddings, err := s.embedding.GenerateBatchEmbeddings(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate batch embeddings: %w", err)
	}

	// Prepare points
	points := make([]Point, len(documents))
	for i, doc := range documents {
		payload := make(map[string]interface{})
		for k, v := range doc.Metadata {
			payload[k] = v
		}
		payload["text"] = doc.Text

		points[i] = Point{
			ID:      doc.ID,
			Vector:  embeddings[i],
			Payload: payload,
		}
	}

	// Upsert all points
	return s.provider.Upsert(ctx, collection, points)
}

// Search performs semantic search
func (s *Service) Search(ctx context.Context, collection, query string, limit int, filter *Filter) ([]SearchResult, error) {
	// Generate query embedding
	queryEmbedding, err := s.embedding.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Perform search
	return s.provider.Search(ctx, collection, queryEmbedding, limit, filter)
}

// DeleteDocument deletes a document by ID
func (s *Service) DeleteDocument(ctx context.Context, collection, documentID string) error {
	return s.provider.Delete(ctx, collection, []string{documentID})
}

// DeleteDocuments deletes multiple documents by IDs
func (s *Service) DeleteDocuments(ctx context.Context, collection string, documentIDs []string) error {
	return s.provider.Delete(ctx, collection, documentIDs)
}

// GetCollectionInfo gets collection information
func (s *Service) GetCollectionInfo(ctx context.Context, collection string) (*CollectionInfo, error) {
	return s.provider.GetCollectionInfo(ctx, collection)
}

// Close closes all connections
func (s *Service) Close() error {
	return s.provider.Close()
}

// GetProviderType returns the current provider type
func (s *Service) GetProviderType() string {
	return s.provider.GetProviderType()
}

// GetEmbeddingModel returns the current embedding model name
func (s *Service) GetEmbeddingModel() string {
	return s.embedding.GetProviderName()
}

// Document represents a document to be added to the vector database
type Document struct {
	ID       string                 // Unique document ID (if empty, UUID will be generated)
	Text     string                 // Document text content
	Metadata map[string]interface{} // Additional metadata
}

// GenerateDocumentID generates a unique UUID for a document
func GenerateDocumentID() string {
	return uuid.New().String()
}

// ChunkText splits a long text into smaller chunks for better search accuracy
// This is useful for long documents that exceed token limits
func ChunkText(text string, maxChunkSize int, overlap int) []string {
	if len(text) <= maxChunkSize {
		return []string{text}
	}

	chunks := []string{}
	start := 0

	for start < len(text) {
		end := start + maxChunkSize
		if end > len(text) {
			end = len(text)
		}

		chunk := text[start:end]
		chunks = append(chunks, chunk)

		// Move start with overlap
		start = end - overlap
		if start < 0 {
			start = 0
		}
	}

	return chunks
}

// ChunkDocuments chunks a single large document into multiple smaller documents
func ChunkDocuments(doc Document, maxChunkSize int, overlap int) []Document {
	chunks := ChunkText(doc.Text, maxChunkSize, overlap)
	documents := make([]Document, len(chunks))

	baseID := doc.ID
	if baseID == "" {
		baseID = GenerateDocumentID()
	}

	for i, chunk := range chunks {
		metadata := make(map[string]interface{})
		for k, v := range doc.Metadata {
			metadata[k] = v
		}
		metadata["chunk_index"] = i
		metadata["total_chunks"] = len(chunks)
		metadata["parent_doc_id"] = baseID

		documents[i] = Document{
			ID:       fmt.Sprintf("%s_chunk_%d", baseID, i),
			Text:     chunk,
			Metadata: metadata,
		}
	}

	return documents
}
