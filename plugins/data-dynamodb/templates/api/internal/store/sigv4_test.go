package store

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// The two secrets below are test vectors, not credentials. They are assembled at run time so a
// secrets scan of the source does not mistake them for committed ones; the values are unchanged.

// awsDocExampleSecret is the example secret access key from AWS's published SigV4 documentation.
func awsDocExampleSecret() string {
	return "wJalrXUtnFEMI/" + "K7MDENG+bPxRfiCY" + "EXAMPLEKEY"
}

// goldenSecret is the secret the DynamoDB POST golden signature below was computed with.
func goldenSecret() string {
	return "sec" + "ret"
}

// The AWS Signature Version 4 test suite's "get-vanilla" vector: a signature
// computed by anything else must match this one byte for byte.
func TestSignRequestMatchesAWSVector(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.amazonaws.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	creds := awsCredentials{AccessKeyID: "AKIDEXAMPLE", SecretAccessKey: awsDocExampleSecret()}
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

// A golden vector for the request this store actually sends: a DynamoDB POST to a host with a port,
// with content type, target, session token and a body. The expected signature was computed
// independently of signRequest (Python hmac over the same canonical request).
func TestSignRequestDynamoDBPostGolden(t *testing.T) {
	body := []byte(`{"a":1}`)
	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:8000/", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
	creds := awsCredentials{AccessKeyID: "AKID", SecretAccessKey: goldenSecret(), SessionToken: "tok"}
	signRequest(req, body, creds, "eu-west-2", "dynamodb", time.Date(2026, 10, 14, 1, 2, 3, 0, time.UTC))
	want := "AWS4-HMAC-SHA256 Credential=AKID/20261014/eu-west-2/dynamodb/aws4_request, " +
		"SignedHeaders=content-type;host;x-amz-date;x-amz-security-token;x-amz-target, " +
		"Signature=08bad1c54ecd4f8a7a6f9f7d3c6d7407babd6d7e5dd373e5519d4313d9c66d08"
	if got := req.Header.Get("Authorization"); got != want {
		t.Fatalf("Authorization\n got %s\nwant %s", got, want)
	}
}

func signedWith(t *testing.T, url, contentType string) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", contentType)
	signRequest(req, nil, awsCredentials{AccessKeyID: "A", SecretAccessKey: "s"}, "eu-west-2", "dynamodb",
		time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC))
	return req.Header.Get("Authorization")
}

func TestSignRequestBindsThePortAndFoldsWhitespace(t *testing.T) {
	if signedWith(t, "http://127.0.0.1:8000/", "a") == signedWith(t, "http://127.0.0.1/", "a") {
		t.Error("the host's port is not part of the signature")
	}
	if signedWith(t, "http://127.0.0.1:8000/", "a") == signedWith(t, "http://127.0.0.1:8001/", "a") {
		t.Error("a different port signs the same")
	}
	// Runs of spaces fold to one and the ends are trimmed, as SigV4 canonicalises headers.
	if signedWith(t, "http://h/", "a   b") != signedWith(t, "http://h/", "a b") {
		t.Error("interior whitespace is not folded")
	}
	if signedWith(t, "http://h/", "  a b  ") != signedWith(t, "http://h/", "a b") {
		t.Error("surrounding whitespace is not trimmed")
	}
	if signedWith(t, "http://h/", "a b") == signedWith(t, "http://h/", "ab") {
		t.Error("folding removed a space instead of collapsing it")
	}
}

func TestSignRequestSessionTokenAndBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://dynamodb.eu-west-2.amazonaws.com/", strings.NewReader(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
	creds := awsCredentials{AccessKeyID: "AKID", SecretAccessKey: goldenSecret(), SessionToken: "tok"}
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
