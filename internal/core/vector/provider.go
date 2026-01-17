package vector

import (
	"context"
)

// Provider defines the interface for vector database operations
// Supports both self-hosted and cloud-based Qdrant instances
type Provider interface {
	// Initialize initializes the connection to vector database
	Initialize(ctx context.Context) error

	// CreateCollection creates a new collection (if not exists)
	CreateCollection(ctx context.Context, name string, vectorSize int) error

	// DeleteCollection deletes a collection
	DeleteCollection(ctx context.Context, name string) error

	// Upsert inserts or updates vectors in a collection
	Upsert(ctx context.Context, collection string, points []Point) error

	// Search performs similarity search
	Search(ctx context.Context, collection string, query []float32, limit int, filter *Filter) ([]SearchResult, error)

	// Delete deletes points by IDs
	Delete(ctx context.Context, collection string, ids []string) error

	// GetCollectionInfo gets information about a collection
	GetCollectionInfo(ctx context.Context, collection string) (*CollectionInfo, error)

	// Close closes the connection
	Close() error

	// GetProviderType returns the provider type ("qdrant_cloud" or "qdrant_self_hosted")
	GetProviderType() string
}

// Point represents a vector point with metadata
type Point struct {
	ID       string                 `json:"id"`
	Vector   []float32              `json:"vector"`
	Payload  map[string]interface{} `json:"payload,omitempty"`
}

// SearchResult represents a search result
type SearchResult struct {
	ID      string                 `json:"id"`
	Score   float32                `json:"score"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// Filter represents search filters
type Filter struct {
	Must   []Condition `json:"must,omitempty"`
	Should []Condition `json:"should,omitempty"`
	MustNot []Condition `json:"must_not,omitempty"`
}

// Condition represents a filter condition
type Condition struct {
	Key      string      `json:"key"`
	Match    interface{} `json:"match,omitempty"`
	Range    *Range      `json:"range,omitempty"`
}

// Range represents a range condition
type Range struct {
	Gte *float64 `json:"gte,omitempty"` // Greater than or equal
	Gt  *float64 `json:"gt,omitempty"`  // Greater than
	Lte *float64 `json:"lte,omitempty"` // Less than or equal
	Lt  *float64 `json:"lt,omitempty"`  // Less than
}

// CollectionInfo represents collection metadata
type CollectionInfo struct {
	Name        string `json:"name"`
	VectorSize  int    `json:"vector_size"`
	PointsCount int64  `json:"points_count"`
	Status      string `json:"status"`
}
