package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// QdrantCloudProvider implements Provider for Qdrant Cloud
type QdrantCloudProvider struct {
	apiKey     string
	url        string
	httpClient *http.Client
}

// NewQdrantCloudProvider creates a new Qdrant Cloud provider
// url format: https://xxx-xxx.us-east.aws.cloud.qdrant.io
func NewQdrantCloudProvider(url, apiKey string) (*QdrantCloudProvider, error) {
	if url == "" {
		return nil, fmt.Errorf("Qdrant Cloud URL is required")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Qdrant Cloud API key is required")
	}

	return &QdrantCloudProvider{
		apiKey: apiKey,
		url:    url,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Initialize initializes the connection to Qdrant Cloud
func (p *QdrantCloudProvider) Initialize(ctx context.Context) error {
	// Test connection by listing collections
	req, err := http.NewRequestWithContext(ctx, "GET", p.url+"/collections", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("api-key", p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Qdrant Cloud: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Qdrant Cloud connection failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreateCollection creates a new collection in Qdrant Cloud
func (p *QdrantCloudProvider) CreateCollection(ctx context.Context, name string, vectorSize int) error {
	payload := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     vectorSize,
			"distance": "Cosine", // Cosine similarity
		},
	}

	return p.doRequest(ctx, "PUT", fmt.Sprintf("/collections/%s", name), payload, nil)
}

// DeleteCollection deletes a collection
func (p *QdrantCloudProvider) DeleteCollection(ctx context.Context, name string) error {
	return p.doRequest(ctx, "DELETE", fmt.Sprintf("/collections/%s", name), nil, nil)
}

// Upsert inserts or updates points
func (p *QdrantCloudProvider) Upsert(ctx context.Context, collection string, points []Point) error {
	// Convert to Qdrant format
	qdrantPoints := make([]map[string]interface{}, len(points))
	for i, point := range points {
		qdrantPoints[i] = map[string]interface{}{
			"id":      point.ID,
			"vector":  point.Vector,
			"payload": point.Payload,
		}
	}

	payload := map[string]interface{}{
		"points": qdrantPoints,
	}

	return p.doRequest(ctx, "PUT", fmt.Sprintf("/collections/%s/points", collection), payload, nil)
}

// Search performs similarity search
func (p *QdrantCloudProvider) Search(ctx context.Context, collection string, query []float32, limit int, filter *Filter) ([]SearchResult, error) {
	payload := map[string]interface{}{
		"vector": query,
		"limit":  limit,
	}

	if filter != nil {
		payload["filter"] = p.convertFilter(filter)
	}

	var response struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float32                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	err := p.doRequest(ctx, "POST", fmt.Sprintf("/collections/%s/points/search", collection), payload, &response)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(response.Result))
	for i, r := range response.Result {
		results[i] = SearchResult{
			ID:      r.ID,
			Score:   r.Score,
			Payload: r.Payload,
		}
	}

	return results, nil
}

// Delete deletes points by IDs
func (p *QdrantCloudProvider) Delete(ctx context.Context, collection string, ids []string) error {
	payload := map[string]interface{}{
		"points": ids,
	}

	return p.doRequest(ctx, "POST", fmt.Sprintf("/collections/%s/points/delete", collection), payload, nil)
}

// GetCollectionInfo gets collection information
func (p *QdrantCloudProvider) GetCollectionInfo(ctx context.Context, collection string) (*CollectionInfo, error) {
	var response struct {
		Result struct {
			Config struct {
				Params struct {
					Vectors struct {
						Size int `json:"size"`
					} `json:"vectors"`
				} `json:"params"`
			} `json:"config"`
			PointsCount int64  `json:"points_count"`
			Status      string `json:"status"`
		} `json:"result"`
	}

	err := p.doRequest(ctx, "GET", fmt.Sprintf("/collections/%s", collection), nil, &response)
	if err != nil {
		return nil, err
	}

	return &CollectionInfo{
		Name:        collection,
		VectorSize:  response.Result.Config.Params.Vectors.Size,
		PointsCount: response.Result.PointsCount,
		Status:      response.Result.Status,
	}, nil
}

// Close closes the connection
func (p *QdrantCloudProvider) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}

// GetProviderType returns the provider type
func (p *QdrantCloudProvider) GetProviderType() string {
	return "qdrant_cloud"
}

// doRequest performs an HTTP request to Qdrant Cloud
func (p *QdrantCloudProvider) doRequest(ctx context.Context, method, path string, payload interface{}, result interface{}) error {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.url+path, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("api-key", p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("request failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// convertFilter converts our Filter format to Qdrant filter format
func (p *QdrantCloudProvider) convertFilter(filter *Filter) map[string]interface{} {
	qdrantFilter := make(map[string]interface{})

	if len(filter.Must) > 0 {
		must := make([]map[string]interface{}, len(filter.Must))
		for i, cond := range filter.Must {
			must[i] = p.convertCondition(cond)
		}
		qdrantFilter["must"] = must
	}

	if len(filter.Should) > 0 {
		should := make([]map[string]interface{}, len(filter.Should))
		for i, cond := range filter.Should {
			should[i] = p.convertCondition(cond)
		}
		qdrantFilter["should"] = should
	}

	if len(filter.MustNot) > 0 {
		mustNot := make([]map[string]interface{}, len(filter.MustNot))
		for i, cond := range filter.MustNot {
			mustNot[i] = p.convertCondition(cond)
		}
		qdrantFilter["must_not"] = mustNot
	}

	return qdrantFilter
}

// convertCondition converts a Condition to Qdrant format
func (p *QdrantCloudProvider) convertCondition(cond Condition) map[string]interface{} {
	condition := map[string]interface{}{
		"key": cond.Key,
	}

	if cond.Match != nil {
		condition["match"] = map[string]interface{}{
			"value": cond.Match,
		}
	}

	if cond.Range != nil {
		rangeFilter := make(map[string]interface{})
		if cond.Range.Gte != nil {
			rangeFilter["gte"] = *cond.Range.Gte
		}
		if cond.Range.Gt != nil {
			rangeFilter["gt"] = *cond.Range.Gt
		}
		if cond.Range.Lte != nil {
			rangeFilter["lte"] = *cond.Range.Lte
		}
		if cond.Range.Lt != nil {
			rangeFilter["lt"] = *cond.Range.Lt
		}
		condition["range"] = rangeFilter
	}

	return condition
}
