package client

import (
	"net/http"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/ratelimit"
)

// requestsPerMinutePerToken is the Snyk REST API limit per token (terraform and other clients share this budget).
const requestsPerMinutePerToken = 1620

// newHTTPClient returns an *http.Client that rate-limits outbound requests and retries transient failures.
func newHTTPClient() *http.Client {
	rl := ratelimit.New(requestsPerMinutePerToken, ratelimit.Per(time.Minute), ratelimit.WithoutSlack)

	rc := retryablehttp.NewClient()
	rc.Logger = nil // provider runs inside Terraform; avoid retry debug noise on stderr

	rc.HTTPClient = &http.Client{
		Transport: &rateLimitTransport{
			limiter: rl,
			base:    cleanhttp.DefaultPooledTransport(),
		},
	}

	return rc.StandardClient()
}

// rateLimitTransport calls the Uber limiter before each underlying RoundTrip (including retryablehttp retries).
type rateLimitTransport struct {
	limiter ratelimit.Limiter
	base    http.RoundTripper
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.limiter.Take()
	return t.base.RoundTrip(req)
}
