// Package dynamodb provides DynamoDB-backed storage for publish status.
package dynamodb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	ttlDuration       = 30 * 24 * time.Hour
	batchGetChunkSize = 100
)

// PublishStatus is a record stored in DynamoDB to track Bluesky posts.
type PublishStatus struct {
	ExpiresAt int64 `dynamodbav:"ExpiresAt"`
	GUID      string
	Published string
	Title     string
}

// Store is a DynamoDB-backed publish-status store.
type Store struct {
	client    *dynamodb.Client
	tableName string
}

// NewStore constructs a Store using the supplied AWS configuration.
func NewStore(awscfg *aws.Config, tableName string) *Store {
	return &Store{
		client:    dynamodb.NewFromConfig(*awscfg),
		tableName: tableName,
	}
}

// StorePublishStatus writes a PublishStatus record with a TTL.
func (db *Store) StorePublishStatus(ctx context.Context, status PublishStatus) error {
	status.ExpiresAt = time.Now().Add(ttlDuration).Unix()

	av, err := attributevalue.MarshalMap(status)
	if err != nil {
		return fmt.Errorf("marshalling publish status: %w", err)
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(db.tableName),
	}

	_, err = db.client.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("putting item to DynamoDB: %w", err)
	}

	return nil
}

// IsPublishedBatch returns a map keyed by GUID whose value is true if a
// PublishStatus record exists. Missing keys are reported as false. Calls
// BatchGetItem in chunks of 100 to stay within DynamoDB's per-request
// limit. UnprocessedKeys are treated as not-published — re-issuing posts
// for them is safe because Bluesky deduplicates by deterministic Rkey.
func (db *Store) IsPublishedBatch(ctx context.Context, guids []string) (map[string]bool, error) {
	result := make(map[string]bool, len(guids))

	for _, guid := range guids {
		result[guid] = false
	}

	for start := 0; start < len(guids); start += batchGetChunkSize {
		end := min(start+batchGetChunkSize, len(guids))

		err := db.fetchBatch(ctx, guids[start:end], result)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (db *Store) fetchBatch(ctx context.Context, chunk []string, result map[string]bool) error {
	keys := make([]map[string]types.AttributeValue, 0, len(chunk))
	for _, guid := range chunk {
		keys = append(keys, map[string]types.AttributeValue{
			"GUID": &types.AttributeValueMemberS{Value: guid},
		})
	}

	out, err := db.client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			db.tableName: {
				Keys:                 keys,
				ProjectionExpression: aws.String("GUID"),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("batch-getting items from DynamoDB: %w", err)
	}

	for _, item := range out.Responses[db.tableName] {
		if attr, ok := item["GUID"].(*types.AttributeValueMemberS); ok {
			result[attr.Value] = true
		}
	}

	return nil
}
