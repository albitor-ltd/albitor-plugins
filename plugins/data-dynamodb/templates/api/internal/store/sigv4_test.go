package store

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// The AWS Signature Version 4 test suite's "get-vanilla" vector: a signature
// computed by anything else must match this one byte for byte.
func TestSignRequestMatchesAWSVector(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.amazonaws.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	creds := awsCredentials{AccessKeyID: "AKIDEXAMPLE", SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"}
	signRequest(req, nil, creds, "us-east-1", "service", time.Date(2015, 8, 30, 12, 36, 0, 0, time.UTC))

	want := "AWS4-HMAC-SHA256 Credential=AKIDEXAMPLE/20150830/us-east-1/service/aws4_request, " +
		"SignedHeaders=host;x-amz-date, " +
		"Signature=5fa00fa31553b73ebf1942676e86291e8372ff2a2260956d9b8aae1d763fbf31"
	if got := req.Header.Get("Authorization"); got != want {
		t.Fatalf("Authorization\n got %s\nwant %s", got, want)
	}
	if got := req.Header.Get("X-Amz-Date"); got != "20150830T123600Z" {
		t.Fatalf("X-Amz-Date = %q", got)
	}
}

func TestSignRequestSessionTokenAndBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://dynamodb.eu-west-2.amazonaws.com/", strings.NewReader(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
	creds := awsCredentials{AccessKeyID: "AKID", SecretAccessKey: "secret", SessionToken: "tok"}
	now := time.Date(2026, 10, 14, 1, 2, 3, 0, time.UTC)
	signRequest(req, []byte(`{"a":1}`), creds, "eu-west-2", "dynamodb", now)

	auth := req.Header.Get("Authorization")
	for _, want := range []string{
		"Credential=AKID/20261014/eu-west-2/dynamodb/aws4_request",
		"SignedHeaders=content-type;host;x-amz-date;x-amz-security-token;x-amz-target",
	} {
		if !strings.Contains(auth, want) {
			t.Errorf("Authorization %q lacks %q", auth, want)
		}
	}
	if strings.Contains(auth, "secret") {
		t.Errorf("the secret key leaked into the header: %s", auth)
	}
	if req.Header.Get("X-Amz-Security-Token") != "tok" {
		t.Errorf("session token not sent")
	}
	// A different body signs differently: the payload hash is in the signature.
	req2, _ := http.NewRequest(http.MethodPost, "https://dynamodb.eu-west-2.amazonaws.com/", nil)
	req2.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req2.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
	signRequest(req2, []byte(`{"a":2}`), creds, "eu-west-2", "dynamodb", now)
	if req2.Header.Get("Authorization") == auth {
		t.Errorf("the signature does not depend on the body")
	}
}
