package store

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	// DataTable is the logical name of the table this plugin declares in
	// albitor.yaml (services.tables). The platform names the AWS table and
	// hands it to the function as ALBITOR_TABLE_DATA.
	DataTable = "data"

	// LocalModeEnv is the local run's switch, shared with the other plugins.
	LocalModeEnv = "ALBITOR_LOCAL"
	// TableEnvPrefix prefixes the environment variable carrying the AWS name of
	// each declared table.
	TableEnvPrefix = "ALBITOR_TABLE_"
	// EndpointEnv points the adapter at DynamoDB Local. Loopback hosts only.
	EndpointEnv = "LOCAL_DYNAMODB_ENDPOINT"

	// lambdaMarker is set by the Lambda runtime itself, so configuration cannot
	// unset it.
	lambdaMarker = "AWS_LAMBDA_FUNCTION_NAME"
)

var logicalRE = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`)

// TableEnv is the variable that names the AWS table for a logical table name.
func TableEnv(logical string) string {
	return TableEnvPrefix + strings.ToUpper(strings.ReplaceAll(logical, "-", "_"))
}

// Open picks the adapter from the environment, read through getenv:
//
//   - ALBITOR_LOCAL=1 is a local run: the in-memory adapter. It is refused on
//     a deployed function and beside a configured table, so a deployed app can
//     never quietly serve from process memory and lose its data.
//   - Otherwise the DynamoDB adapter, which needs ALBITOR_TABLE_<NAME>, the
//     region and the execution role's credentials, and refuses to start
//     without them. Missing AWS configuration never selects the fake.
func Open(getenv func(string) string, logical string) (Store, error) {
	if !logicalRE.MatchString(logical) {
		return nil, fmt.Errorf("store: %q is not a table name", logical)
	}
	env := func(name string) string { return strings.TrimSpace(getenv(name)) }
	table := env(TableEnv(logical))

	if env(LocalModeEnv) == "1" {
		if env(lambdaMarker) != "" {
			return nil, errors.New("store: ALBITOR_LOCAL=1 on a deployed function; refusing to serve from memory")
		}
		if table != "" {
			return nil, fmt.Errorf("store: ALBITOR_LOCAL=1 beside %s; a local run uses no table", TableEnv(logical))
		}
		return NewMemory(), nil
	}
	if table == "" {
		return nil, fmt.Errorf("store: %s is not set; the app cannot start without its table", TableEnv(logical))
	}
	// A deployed function always talks to the regional endpoint: nothing in its
	// environment may steer signed requests elsewhere, loopback or not.
	endpoint := env(EndpointEnv)
	if env(lambdaMarker) != "" {
		endpoint = ""
	}
	return NewDynamoDB(DynamoDBConfig{
		Table:           table,
		Region:          env("AWS_REGION"),
		Endpoint:        endpoint,
		AccessKeyID:     env("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: env("AWS_SECRET_ACCESS_KEY"),
		SessionToken:    env("AWS_SESSION_TOKEN"),
	})
}
