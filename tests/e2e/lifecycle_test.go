//go:build e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

func testBaseURL(t *testing.T) string {
	t.Helper()
	base := os.Getenv("FANPULSE_TEST_BASE_URL")
	if base == "" {
		t.Fatal("FANPULSE_TEST_BASE_URL is required for e2e tests")
	}
	return base
}

func TestHealthEndpoints(t *testing.T) {
	client := &http.Client{Timeout: 2 * time.Second}
	for _, path := range []string{"/health/live", "/health/ready"} {
		resp, err := client.Get(testBaseURL(t) + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		var body struct {
			Status string `json:"status"`
		}
		err = json.NewDecoder(resp.Body).Decode(&body)
		_ = resp.Body.Close()
		if err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		if resp.StatusCode != http.StatusOK || body.Status != "ok" {
			t.Fatalf("%s status=%d body=%#v", path, resp.StatusCode, body)
		}
	}
}
