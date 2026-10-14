package shell

import (
	"fmt"
	"log/slog"
	"net/http"

	"webapp/internal/localauth"
)

// Boot returns the handler a process serves: the composed router, with the
// local test issuer wrapped round it when a local run asks for one.
//
// A local run (ALBITOR_LOCAL=1 ALBITOR_LOCAL_AUTH=1) starts an in-process
// stand-in for the user pool, so specs sign in as named test identities with
// no sign-up, no e-mail and no call to AWS. It only issues tokens, and it
// refuses to start on a deployed function or beside a configured user pool
// (localauth.Open). Whether the API verifies those tokens is the auth
// plugin's middleware, not the shell's.
func Boot(getenv func(string) string, logger *slog.Logger) (http.Handler, error) {
	handler, err := Handler(logger)
	if err != nil {
		return nil, fmt.Errorf("build the router: %w", err)
	}
	issuer, err := localauth.Open(getenv)
	if err != nil {
		return nil, fmt.Errorf("start the local issuer: %w", err)
	}
	if issuer != nil {
		handler = issuer.Wrap(handler)
		logger.Warn("local issuer running: test tokens are minted at "+localauth.Path+"/token",
			slog.String("issuer", issuer.URL()))
	}
	return handler, nil
}
