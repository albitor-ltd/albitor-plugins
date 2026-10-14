package store_test

// Copied into internal/store by scripts/verify-data-dynamodb.sh. It is not part of the managed tree
// an app receives, so an app's own CI needs no Docker. It fails, never skips, when no DynamoDB Local
// endpoint is configured: a conformance run that quietly tests nothing is worse than none.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"webapp/internal/store"
	"webapp/internal/store/storetest"
)

func TestDynamoDBLocalConformance(t *testing.T) {
	endpoint := os.Getenv("DYNAMODB_LOCAL_ENDPOINT")
	if endpoint == "" {
		t.Fatal("DYNAMODB_LOCAL_ENDPOINT is not set; scripts/verify-data-dynamodb.sh sets it")
	}
	table := "conformance-" + time.Now().Format("150405")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := createTable(ctx, endpoint, table); err != nil {
		t.Fatal(err)
	}
	storetest.Run(t, func(t *testing.T) store.Store {
		s, err := store.NewDynamoDB(store.DynamoDBConfig{
			Table: table, Region: "eu-west-2", Endpoint: endpoint,
			AccessKeyID: "local", SecretAccessKey: "local",
		})
		if err != nil {
			t.Fatal(err)
		}
		return s
	})
}

// createTable makes the one-table layout the Terraform module gives a deployed app: string pk and
// sk, on demand. DynamoDB Local does not check signatures, so a bare Authorization header names a
// region and key.
func createTable(ctx context.Context, endpoint, table string) error {
	body := fmt.Sprintf(`{"TableName":%q,"BillingMode":"PAY_PER_REQUEST",`+
		`"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"},{"AttributeName":"sk","AttributeType":"S"}],`+
		`"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"},{"AttributeName":"sk","KeyType":"RANGE"}]}`, table)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/", strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.CreateTable")
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=local/20260101/eu-west-2/dynamodb/aws4_request, SignedHeaders=host, Signature=x")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
		return fmt.Errorf("create table: HTTP %d: %s", resp.StatusCode, b)
	}
	return nil
}
