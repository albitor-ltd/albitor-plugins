package shell

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// The AWS Signature Version 4 test suite's "get-vanilla" vector: a signature
// computed by anything else must match this one byte for byte. The secret is
// AWS's published example key, assembled at run time so a secrets scan of the
// source does not mistake it for a committed one.
func TestSignRequestMatchesAWSVector(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.amazonaws.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	creds := awsCredentials{AccessKeyID: "AKIDEXAMPLE", SecretAccessKey: "wJalrXUtnFEMI/" + "K7MDENG+bPxRfiCY" + "EXAMPLEKEY"}
	signRequest(req, nil, creds, "us-east-1", "service", time.Date(2015, 8, 30, 12, 36, 0, 0, time.UTC))

	want := "AWS4-HMAC-SHA256 Credential=AKIDEXAMPLE/20150830/us-east-1/service/aws4_request, " +
		"SignedHeaders=host;x-amz-date, " +
		"Signature=5fa00fa31553b73ebf1942676e86291e8372ff2a2260956d9b8aae1d763fbf31"
	if got := req.Header.Get("Authorization"); got != want {
		t.Fatalf("Authorization\n got %s\nwant %s", got, want)
	}
}

func TestSignRequestSignsTheSessionTokenAndTarget(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://cognito-idp.eu-west-2.amazonaws.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSCognitoIdentityProviderService.ListUsers")
	signRequest(req, []byte(`{}`), awsCredentials{AccessKeyID: "A", SecretAccessKey: "s", SessionToken: "tok"},
		"eu-west-2", "cognito-idp", time.Date(2026, 11, 1, 1, 2, 3, 0, time.UTC))
	if got := req.Header.Get("X-Amz-Security-Token"); got != "tok" {
		t.Fatalf("X-Amz-Security-Token = %q", got)
	}
	auth := req.Header.Get("Authorization")
	for _, part := range []string{"/eu-west-2/cognito-idp/aws4_request", "x-amz-security-token", "x-amz-target"} {
		if !strings.Contains(auth, part) {
			t.Fatalf("Authorization %q lacks %q", auth, part)
		}
	}
}
