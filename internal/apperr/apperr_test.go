package apperr

import "testing"

func TestAppErrorBuilderAndError(t *testing.T) {
	err := New("RATE_LIMITED", "retry later").
		WithRetryable(true).
		WithHTTPStatus(429).
		WithUpstreamCode("too_many_requests")

	if err.Code != "RATE_LIMITED" {
		t.Fatalf("unexpected code: %q", err.Code)
	}
	if err.Error() != "retry later" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
	if !err.Retryable || err.HTTPStatus != 429 || err.UpstreamCode != "too_many_requests" {
		t.Fatalf("unexpected app error fields: %+v", err)
	}
}
