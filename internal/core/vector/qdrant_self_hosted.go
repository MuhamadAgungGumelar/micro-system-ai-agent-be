package vector

import (
	"context"
	"fmt"
	"log"

	qdrant "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// QdrantSelfHostedProvider implements Provider for self-hosted Qdrant
type QdrantSelfHostedProvider struct {
	host       string
	port       int
	grpcConn   *grpc.ClientConn
	client     qdrant.PointsClient
	collection qdrant.CollectionsClient
}

// NewQdrantSelfHostedProvider creates a new self-hosted Qdrant provider
// Default: host="localhost", port=6334 (gRPC port)
func NewQdrantSelfHostedProvider(host string, port int) (*QdrantSelfHostedProvider, error) {
	if host == "" {
		host = "localhost"
	}
	if port == 0 {
		port = 6334 // Default gRPC port
	}

	return &QdrantSelfHostedProvider{
		host: host,
		port: port,
	}, nil
}

// Initialize initializes the connection to self-hosted Qdrant
func (p *QdrantSelfHostedProvider) Initialize(ctx context.Context) error {
	address := fmt.Sprintf("%s:%d", p.host, p.port)
	log.Printf("🔗 Connecting to Qdrant at %s...", address)

	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	p.grpcConn = conn
	p.client = qdrant.NewPointsClient(conn)
	p.collection = qdrant.NewCollectionsClient(conn)

	log.Printf("✅ Connected to Qdrant successfully")
	return nil
}

// CreateCollection creates a new collection
func (p *QdrantSelfHostedProvider) CreateCollection(ctx context.Context, name string, vectorSize int) error {
	// Check if collection exists
	exists, err := p.collectionExists(ctx, name)
	if err != nil {
		return err
	}

	if exists {
		log.Printf("⚠️  Collection '%s' already exists", name)
		return nil
	}

	// Create collection
	_, err = p.collection.Create(ctx, &qdrant.CreateCollection{
		CollectionName: name,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     uint64(vectorSize),
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	log.Printf("✅ Collection '%s' created", name)
	return nil
}

// DeleteCollection deletes a collection
func (p *QdrantSelfHostedProvider) DeleteCollection(ctx context.Context, name string) error {
	_, err := p.collection.Delete(ctx, &qdrant.DeleteCollection{
		CollectionName: name,
	})

	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	log.Printf("✅ Collection '%s' deleted", name)
	return nil
}

// Upsert inserts or updates points
func (p *QdrantSelfHostedProvider) Upsert(ctx context.Context, collection string, points []Point) error {
	qdrantPoints := make([]*qdrant.PointStruct, len(points))

	for i, point := range points {
		// Convert payload to Qdrant format
		payload := make(map[string]*qdrant.Value)
		for key, val := range point.Payload {
			payload[key] = convertToQdrantValue(val)
		}

		qdrantPoints[i] = &qdrant.PointStruct{
			Id: &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Uuid{
					Uuid: point.ID,
				},
			},
			Vectors: &qdrant.Vectors{
				VectorsOptions: &qdrant.Vectors_Vector{
					Vector: &qdrant.Vector{
						Data: point.Vector,
					},
				},
			},
			Payload: payload,
		}
	}

	_, err := p.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collection,
		Points:         qdrantPoints,
	})

	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}

	return nil
}

// Search performs similarity search
func (p *QdrantSelfHostedProvider) Search(ctx context.Context, collection string, query []float32, limit int, filter *Filter) ([]SearchResult, error) {
	searchParams := &qdrant.SearchPoints{
		CollectionName: collection,
		Vector:         query,
		Limit:          uint64(limit),
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
	}

	// Add filter if provided
	if filter != nil {
		searchParams.Filter = p.convertFilter(filter)
	}

	response, err := p.client.Search(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	results := make([]SearchResult, len(response.Result))
	for i, hit := range response.Result {
		payload := make(map[string]interface{})
		for key, val := range hit.Payload {
			payload[key] = convertFromQdrantValue(val)
		}

		results[i] = SearchResult{
			ID:      hit.Id.GetUuid(),
			Score:   hit.Score,
			Payload: payload,
		}
	}

	return results, nil
}

// Delete deletes points by IDs
func (p *QdrantSelfHostedProvider) Delete(ctx context.Context, collection string, ids []string) error {
	pointIDs := make([]*qdrant.PointId, len(ids))
	for i, id := range ids {
		pointIDs[i] = &qdrant.PointId{
			PointIdOptions: &qdrant.PointId_Uuid{
				Uuid: id,
			},
		}
	}

	_, err := p.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: collection,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: pointIDs,
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete points: %w", err)
	}

	return nil
}

// GetCollectionInfo gets collection information
func (p *QdrantSelfHostedProvider) GetCollectionInfo(ctx context.Context, collection string) (*CollectionInfo, error) {
	response, err := p.collection.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: collection,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get collection info: %w", err)
	}

	result := response.Result
	vectorSize := 0
	if params := result.Config.Params.VectorsConfig.GetParams(); params != nil {
		vectorSize = int(params.Size)
	}

	pointsCount := int64(0)
	if result.PointsCount != nil {
		pointsCount = int64(*result.PointsCount)
	}

	return &CollectionInfo{
		Name:        collection,
		VectorSize:  vectorSize,
		PointsCount: pointsCount,
		Status:      result.Status.String(),
	}, nil
}

// Close closes the gRPC connection
func (p *QdrantSelfHostedProvider) Close() error {
	if p.grpcConn != nil {
		return p.grpcConn.Close()
	}
	return nil
}

// GetProviderType returns the provider type
func (p *QdrantSelfHostedProvider) GetProviderType() string {
	return "qdrant_self_hosted"
}

// collectionExists checks if a collection exists
func (p *QdrantSelfHostedProvider) collectionExists(ctx context.Context, name string) (bool, error) {
	response, err := p.collection.List(ctx, &qdrant.ListCollectionsRequest{})
	if err != nil {
		return false, err
	}

	for _, collection := range response.Collections {
		if collection.Name == name {
			return true, nil
		}
	}

	return false, nil
}

// convertFilter converts our Filter to Qdrant filter
func (p *QdrantSelfHostedProvider) convertFilter(filter *Filter) *qdrant.Filter {
	qdrantFilter := &qdrant.Filter{}

	if len(filter.Must) > 0 {
		must := make([]*qdrant.Condition, len(filter.Must))
		for i, cond := range filter.Must {
			must[i] = p.convertCondition(cond)
		}
		qdrantFilter.Must = must
	}

	if len(filter.Should) > 0 {
		should := make([]*qdrant.Condition, len(filter.Should))
		for i, cond := range filter.Should {
			should[i] = p.convertCondition(cond)
		}
		qdrantFilter.Should = should
	}

	if len(filter.MustNot) > 0 {
		mustNot := make([]*qdrant.Condition, len(filter.MustNot))
		for i, cond := range filter.MustNot {
			mustNot[i] = p.convertCondition(cond)
		}
		qdrantFilter.MustNot = mustNot
	}

	return qdrantFilter
}

// convertCondition converts a Condition to Qdrant condition
func (p *QdrantSelfHostedProvider) convertCondition(cond Condition) *qdrant.Condition {
	condition := &qdrant.Condition{
		ConditionOneOf: &qdrant.Condition_Field{
			Field: &qdrant.FieldCondition{
				Key: cond.Key,
			},
		},
	}

	fieldCond := condition.GetField()

	if cond.Match != nil {
		fieldCond.Match = &qdrant.Match{
			MatchValue: &qdrant.Match_Keyword{
				Keyword: fmt.Sprintf("%v", cond.Match),
			},
		}
	}

	if cond.Range != nil {
		rangeFilter := &qdrant.Range{}
		if cond.Range.Gte != nil {
			rangeFilter.Gte = cond.Range.Gte
		}
		if cond.Range.Gt != nil {
			rangeFilter.Gt = cond.Range.Gt
		}
		if cond.Range.Lte != nil {
			rangeFilter.Lte = cond.Range.Lte
		}
		if cond.Range.Lt != nil {
			rangeFilter.Lt = cond.Range.Lt
		}
		fieldCond.Range = rangeFilter
	}

	return condition
}

// Helper functions to convert between our format and Qdrant format

func convertToQdrantValue(val interface{}) *qdrant.Value {
	switch v := val.(type) {
	case string:
		return &qdrant.Value{
			Kind: &qdrant.Value_StringValue{StringValue: v},
		}
	case int:
		return &qdrant.Value{
			Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(v)},
		}
	case int64:
		return &qdrant.Value{
			Kind: &qdrant.Value_IntegerValue{IntegerValue: v},
		}
	case float64:
		return &qdrant.Value{
			Kind: &qdrant.Value_DoubleValue{DoubleValue: v},
		}
	case bool:
		return &qdrant.Value{
			Kind: &qdrant.Value_BoolValue{BoolValue: v},
		}
	default:
		return &qdrant.Value{
			Kind: &qdrant.Value_StringValue{StringValue: fmt.Sprintf("%v", v)},
		}
	}
}

func convertFromQdrantValue(val *qdrant.Value) interface{} {
	switch v := val.Kind.(type) {
	case *qdrant.Value_StringValue:
		return v.StringValue
	case *qdrant.Value_IntegerValue:
		return v.IntegerValue
	case *qdrant.Value_DoubleValue:
		return v.DoubleValue
	case *qdrant.Value_BoolValue:
		return v.BoolValue
	default:
		return nil
	}
}
