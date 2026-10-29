// Package albitorsvc is the thin client for the Albitor service gateway, the
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
// This package is managed: do not edit it. Declare what the app calls in
// albitor.yaml under services.callable, and use Client from api/internal/app.
package albitorsvc
