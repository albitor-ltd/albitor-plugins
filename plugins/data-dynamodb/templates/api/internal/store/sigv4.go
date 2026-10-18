package store

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
	"time"
)

// awsCredentials are the function's execution-role credentials, as the Lambda
// runtime puts them in the environment. SessionToken is empty for a long-lived
// key (DynamoDB Local accepts any).
type awsCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

// signedHeaders are the headers a request is signed over, when present. Host
// and X-Amz-Date are always signed.
var signedHeaders = []string{"content-type", "host", "x-amz-date", "x-amz-security-token", "x-amz-target"}

// signRequest adds AWS Signature Version 4 to req: X-Amz-Date, X-Amz-Security-
// Token when there is a session token, and Authorization. The standard library
// is enough for one signed POST, which keeps the generated module free of any
// dependency.
//
// req.URL must carry no query string, which is all DynamoDB's JSON API needs.
func signRequest(req *http.Request, body []byte, c awsCredentials, region, service string, now time.Time) {
	now = now.UTC()
	amzDate := now.Format("20060102T150405Z")
	day := now.Format("20060102")

	req.Header.Set("X-Amz-Date", amzDate)
	if c.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", c.SessionToken)
	}

	host := req.Host
	if host == "" {
		host = req.URL.Host
	}
	var names []string
	var canonical strings.Builder
	for _, name := range signedHeaders {
		var value string
		if name == "host" {
			value = host
		} else {
			value = req.Header.Get(name)
		}
		if value == "" {
			continue
		}
		names = append(names, name)
		canonical.WriteString(name + ":" + strings.Join(strings.Fields(value), " ") + "\n")
	}
	sort.Strings(names) // signedHeaders is already sorted; the signature depends on it, so be explicit
	signed := strings.Join(names, ";")

	path := req.URL.EscapedPath()
	if path == "" {
		path = "/"
	}
	canonicalRequest := strings.Join([]string{
		req.Method, path, "", canonical.String(), signed, hexSHA256(body),
	}, "\n")

	scope := day + "/" + region + "/" + service + "/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + hexSHA256([]byte(canonicalRequest))

	key := hmacSHA256([]byte("AWS4"+c.SecretAccessKey), day)
	key = hmacSHA256(key, region)
	key = hmacSHA256(key, service)
	key = hmacSHA256(key, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(key, stringToSign))

	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+c.AccessKeyID+"/"+scope+
		", SignedHeaders="+signed+", Signature="+signature)
}

func hexSHA256(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}
