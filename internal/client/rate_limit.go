package client

import (
	"bytes"
	"io"
	"net/http"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
			base: &loggingTransport{
				base: cleanhttp.DefaultPooledTransport(),
			},
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

// loggingTransport writes HTTP request/response details to terraform-plugin-log.
type loggingTransport struct {
	base http.RoundTripper
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// buffer the request body to stay compatible with retryablehttp per-attempt body handling.
	var requestBody string
	if req.Body != nil {
		reqBody, readErr := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
		if len(reqBody) > 0 {
			requestBody = string(reqBody)
		}
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		tflog.Error(req.Context(), "Snyk API call failed", map[string]any{
			"method":      req.Method,
			"request_url": req.URL.String(),
			"error":       err.Error(),
		})
		return nil, err
	}

	tflog.Debug(req.Context(), "Snyk API call completed", map[string]any{
		"method":      req.Method,
		"request_url": req.URL.String(),
		"status_code": resp.StatusCode,
	})

	if resp.Body == nil {
		return resp, nil
	}

	bodyBytes, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	resp.ContentLength = int64(len(bodyBytes))

	if readErr != nil {
		tflog.Error(req.Context(), "Failed to read Snyk API response body for trace log", map[string]any{
			"method":      req.Method,
			"request_url": req.URL.String(),
			"error":       readErr.Error(),
		})
		return resp, nil
	}

	traceFields := map[string]any{
		"method":        req.Method,
		"request_url":   req.URL.String(),
		"response_body": string(bodyBytes),
	}
	if requestBody != "" {
		traceFields["request_body"] = requestBody
	}
	tflog.Trace(req.Context(), "Snyk API response body", traceFields)

	return resp, nil
}
