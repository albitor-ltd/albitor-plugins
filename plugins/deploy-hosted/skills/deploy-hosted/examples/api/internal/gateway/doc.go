// Package gateway is the thin client for the hosted service gateway, the
// keyless, credit-metered way a hosted app calls AWS AI services.
//
// The app calls POST <ALBITOR_SERVICES_URL>/v1/<service>/<Operation> with the
// AWS API's own JSON request, signed with SigV4 for service "lambda" using the
// function's execution-role credentials. There is no token or secret to hold.
// The gateway rebuilds the request from an allow-list, so a field it does not
// know is refused with 400 bad_request, and a service the manifest does not
// declare under services.callable is refused with 403 not_declared.
//
// A stock AWS SDK client cannot be pointed at the gateway: it signs for the
// target service's name (translate, transcribe), which the function URL cannot
// verify. This package is the fix, and it needs only the standard library, so it
// adds nothing to the module's go.mod. The request and response types are the
// app's own structs, written to AWS's documented JSON field names.
//
// Declare what the app calls in albitor.yaml under services.callable, and use Client from
// api/internal/app. Keep the signing as it is.
//
// queue.go is the queue half: Send to the app's own SQS queues with the function role,
// ParseEvent and WriteResult for the POST /events consumer, and an in-process queue under
// ALBITOR_LOCAL=1. It does not use the gateway.
package gateway
