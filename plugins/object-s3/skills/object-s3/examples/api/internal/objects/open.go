package objects

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// LocalModeEnv is the local run's switch, shared with the rest of the app.
	LocalModeEnv = "ALBITOR_LOCAL"
	// BucketEnvPrefix prefixes the variable carrying the AWS name of each
	// declared bucket: bucket "uploads" is ALBITOR_BUCKET_UPLOADS.
	BucketEnvPrefix = "ALBITOR_BUCKET_"

	// PathEnvPrefix prefixes the variable carrying the path the app's origin
	// serves each bucket at: bucket "uploads" is ALBITOR_OBJECTS_PATH_UPLOADS,
	// "/_objects/uploads/".
	PathEnvPrefix = "ALBITOR_OBJECTS_PATH_"

	lambdaMarker = "AWS_LAMBDA_FUNCTION_NAME"
)

// BucketEnv is the variable that names the AWS bucket for a logical name.
func BucketEnv(logical string) string {
	return BucketEnvPrefix + strings.ToUpper(strings.ReplaceAll(logical, "-", "_"))
}

// PathEnv is the variable that names the origin path for a logical name.
func PathEnv(logical string) string {
	return PathEnvPrefix + strings.ToUpper(strings.ReplaceAll(logical, "-", "_"))
}

// Open picks the store from the environment, read through getenv:
//
//   - ALBITOR_LOCAL=1 is a local run: process memory. It is refused on a
//     deployed function, so a deployed app can never quietly keep files in
//     memory and lose them.
//   - Otherwise S3, which needs ALBITOR_BUCKET_<NAME>, AWS_REGION and the
//     ALBITOR_OBJECTS_PATH_<NAME>, AWS_REGION and the execution role's
//     credentials, and returns an error without them. The caller decides what
//     that means; the example answers 503 at the file routes (Unavailable)
//     and keeps the rest of the app up.
func Open(getenv func(string) string, logical string) (Store, error) {
	env := func(name string) string { return strings.TrimSpace(getenv(name)) }
	if env(LocalModeEnv) == "1" {
		if env(lambdaMarker) != "" {
			return nil, errors.New("objects: ALBITOR_LOCAL=1 on a deployed function; refusing to keep files in memory")
		}
		return NewMemory(), nil
	}
	bucket := env(BucketEnv(logical))
	if bucket == "" {
		return nil, fmt.Errorf("objects: %s is not set; declare services.objects %q in albitor.yaml", BucketEnv(logical), logical)
	}
	path := env(PathEnv(logical))
	if !strings.HasPrefix(path, "/_objects/") || !strings.HasSuffix(path, "/") {
		return nil, fmt.Errorf("objects: %s is %q, want /_objects/<name>/", PathEnv(logical), path)
	}
	return NewS3(S3Config{
		Bucket: bucket,
		Prefix: strings.TrimPrefix(path, "/"),
		Region: env("AWS_REGION"),
		Credentials: Credentials{
			AccessKeyID:     env("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: env("AWS_SECRET_ACCESS_KEY"),
			SessionToken:    env("AWS_SESSION_TOKEN"),
		},
	})
}
