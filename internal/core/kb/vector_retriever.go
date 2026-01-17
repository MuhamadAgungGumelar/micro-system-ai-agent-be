package kb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/vector"
	"github.com/google/uuid"
)

// VectorRetriever provides semantic search for knowledge base using vector database
type VectorRetriever struct {
	vectorService *vector.Service
	collection    string
}

// NewVectorRetriever creates a new vector-powered retriever
func NewVectorRetriever(vectorService *vector.Service, collection string) *VectorRetriever {
	return &VectorRetriever{
		vectorService: vectorService,
		collection:    collection,
	}
}

// Initialize initializes the vector collection for knowledge base
func (r *VectorRetriever) Initialize(ctx context.Context) error {
	log.Printf("🔍 Initializing Vector KB collection: %s", r.collection)

	// Create collection if doesn't exist
	if err := r.vectorService.CreateCollection(ctx, r.collection); err != nil {
		// Collection might already exist, check info
		info, infoErr := r.vectorService.GetCollectionInfo(ctx, r.collection)
		if infoErr != nil {
			return fmt.Errorf("failed to create/verify collection: %w", err)
		}
		log.Printf("✅ Collection '%s' already exists (%d points)", r.collection, info.PointsCount)
	} else {
		log.Printf("✅ Collection '%s' created", r.collection)
	}

	return nil
}

// AddDocument adds a knowledge base document to the vector database
func (r *VectorRetriever) AddDocument(ctx context.Context, clientID, docType, docID, text string, metadata map[string]interface{}) error {
	// Prepare document metadata
	docMetadata := map[string]interface{}{
		"client_id": clientID,
		"doc_type":  docType, // "faq", "product", "policy", "document"
		"doc_id":    docID,
	}

	// Merge user-provided metadata
	for k, v := range metadata {
		docMetadata[k] = v
	}

	// Generate unique ID
	vectorID := fmt.Sprintf("%s_%s_%s", clientID, docType, docID)

	// Add to vector database
	return r.vectorService.AddDocument(ctx, r.collection, vectorID, text, docMetadata)
}

// AddFAQ adds an FAQ to the knowledge base
func (r *VectorRetriever) AddFAQ(ctx context.Context, clientID, faqID, question, answer string) error {
	// Combine question and answer for better semantic search
	text := fmt.Sprintf("Q: %s\nA: %s", question, answer)

	metadata := map[string]interface{}{
		"question": question,
		"answer":   answer,
	}

	return r.AddDocument(ctx, clientID, "faq", faqID, text, metadata)
}

// AddProduct adds a product to the knowledge base
func (r *VectorRetriever) AddProduct(ctx context.Context, clientID, productID, name, description string, price float64, metadata map[string]interface{}) error {
	// Create searchable text from product info
	text := fmt.Sprintf("Product: %s\nDescription: %s\nPrice: %.2f", name, description, price)

	productMetadata := map[string]interface{}{
		"name":        name,
		"description": description,
		"price":       price,
	}

	// Merge additional metadata
	for k, v := range metadata {
		productMetadata[k] = v
	}

	return r.AddDocument(ctx, clientID, "product", productID, text, productMetadata)
}

// Search performs semantic search in the knowledge base
func (r *VectorRetriever) Search(ctx context.Context, clientID, query string, limit int) ([]SearchResult, error) {
	// Create filter for client-specific search
	filter := &vector.Filter{
		Must: []vector.Condition{
			{
				Key:   "client_id",
				Match: clientID,
			},
		},
	}

	// Perform vector search
	results, err := r.vectorService.Search(ctx, r.collection, query, limit, filter)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Convert to KB search results
	kbResults := make([]SearchResult, len(results))
	for i, result := range results {
		kbResults[i] = SearchResult{
			Score:    result.Score,
			Text:     getStringFromPayload(result.Payload, "text"),
			DocType:  getStringFromPayload(result.Payload, "doc_type"),
			DocID:    getStringFromPayload(result.Payload, "doc_id"),
			Metadata: result.Payload,
		}
	}

	return kbResults, nil
}

// SearchByType performs semantic search filtered by document type
func (r *VectorRetriever) SearchByType(ctx context.Context, clientID, query, docType string, limit int) ([]SearchResult, error) {
	filter := &vector.Filter{
		Must: []vector.Condition{
			{
				Key:   "client_id",
				Match: clientID,
			},
			{
				Key:   "doc_type",
				Match: docType,
			},
		},
	}

	results, err := r.vectorService.Search(ctx, r.collection, query, limit, filter)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	kbResults := make([]SearchResult, len(results))
	for i, result := range results {
		kbResults[i] = SearchResult{
			Score:    result.Score,
			Text:     getStringFromPayload(result.Payload, "text"),
			DocType:  getStringFromPayload(result.Payload, "doc_type"),
			DocID:    getStringFromPayload(result.Payload, "doc_id"),
			Metadata: result.Payload,
		}
	}

	return kbResults, nil
}

// DeleteDocument removes a document from the vector database
func (r *VectorRetriever) DeleteDocument(ctx context.Context, clientID, docType, docID string) error {
	vectorID := fmt.Sprintf("%s_%s_%s", clientID, docType, docID)
	return r.vectorService.DeleteDocument(ctx, r.collection, vectorID)
}

// GetRelevantContext retrieves relevant context for LLM from vector search
func (r *VectorRetriever) GetRelevantContext(ctx context.Context, clientID, userQuery string, maxResults int) (string, error) {
	results, err := r.Search(ctx, clientID, userQuery, maxResults)
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "", nil
	}

	// Format results into context string
	context := "Relevant information from knowledge base:\n\n"
	for i, result := range results {
		// Only include high-confidence results (score > 0.7)
		if result.Score < 0.7 {
			continue
		}

		switch result.DocType {
		case "faq":
			question := getStringFromPayload(result.Metadata, "question")
			answer := getStringFromPayload(result.Metadata, "answer")
			context += fmt.Sprintf("%d. Q: %s\n   A: %s\n\n", i+1, question, answer)

		case "product":
			name := getStringFromPayload(result.Metadata, "name")
			description := getStringFromPayload(result.Metadata, "description")
			price := result.Metadata["price"]
			context += fmt.Sprintf("%d. Product: %s\n   Description: %s\n   Price: %v\n\n", i+1, name, description, price)

		default:
			context += fmt.Sprintf("%d. %s (Score: %.2f)\n\n", i+1, result.Text, result.Score)
		}
	}

	return context, nil
}

// SyncFromDatabase syncs knowledge base from PostgreSQL to vector database
// This is useful for initial migration or periodic sync
func (r *VectorRetriever) SyncFromDatabase(ctx context.Context, dbRetriever *Retriever, clientID string) error {
	log.Printf("🔄 Syncing KB from database to vector DB for client: %s", clientID)

	// Get knowledge base from database
	kb, err := dbRetriever.GetKnowledgeBase(clientID)
	if err != nil {
		return fmt.Errorf("failed to get KB from database: %w", err)
	}

	// Sync FAQs
	for i, faq := range kb.FAQs {
		faqID := uuid.New().String()
		if err := r.AddFAQ(ctx, clientID, faqID, faq.Question, faq.Answer); err != nil {
			log.Printf("⚠️  Failed to sync FAQ %d: %v", i, err)
		}
	}
	log.Printf("✅ Synced %d FAQs", len(kb.FAQs))

	// Sync Products
	for i, product := range kb.Products {
		productID := uuid.New().String()
		if err := r.AddProduct(ctx, clientID, productID, product.Name, "", product.Price, nil); err != nil {
			log.Printf("⚠️  Failed to sync product %d: %v", i, err)
		}
	}
	log.Printf("✅ Synced %d products", len(kb.Products))

	return nil
}

// SearchResult represents a knowledge base search result
type SearchResult struct {
	Score    float32                // Similarity score (0-1)
	Text     string                 // Full text content
	DocType  string                 // Document type (faq, product, etc.)
	DocID    string                 // Document ID
	Metadata map[string]interface{} // Additional metadata
}

// Helper function to extract string from payload
func getStringFromPayload(payload map[string]interface{}, key string) string {
	if val, ok := payload[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// Helper function to convert map to JSON string
func toJSONString(data interface{}) string {
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(bytes)
}
