package coursechecks

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func root(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot find repository root")
	}
	return filepath.Dir(filepath.Dir(file))
}

func readRequired(t *testing.T, relative string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root(t), filepath.FromSlash(relative)))
	if err != nil {
		t.Fatalf("required file %s: %v", relative, err)
	}
	return string(data)
}

func requireContains(t *testing.T, relative string, terms ...string) {
	t.Helper()
	content := strings.ToLower(readRequired(t, relative))
	for _, term := range terms {
		if !strings.Contains(content, strings.ToLower(term)) {
			t.Errorf("%s must contain %q", relative, term)
		}
	}
}

func TestLevel00RepositoryBaseline(t *testing.T) {
	for _, file := range []string{"README.md", "go.mod", ".env.example", "docs/architecture.md", "challenges/README.md"} {
		if _, err := os.Stat(filepath.Join(root(t), filepath.FromSlash(file))); err != nil {
			t.Errorf("%s: %v", file, err)
		}
	}
}

func TestLevel04OpenAPIContract(t *testing.T) {
	requireContains(t, "api/openapi/fanpulse.yaml", "openapi: 3.1", "/api/v1/events", "/health/live", "Problem")
}

func TestLevel05Migrations(t *testing.T) {
	requireContains(t, "migrations/000001_initial_schema.up.sql", "create table users", "create table events", "create table event_participants", "unique")
	requireContains(t, "migrations/000001_initial_schema.down.sql", "drop table")
	requireContains(t, "docs/adr/0001-use-postgresql.md", "context", "decision", "consequences")
}

func TestLevel08RedisArtifacts(t *testing.T) {
	requireContains(t, "internal/storage/redis/claim.lua", "redis.call", "stock")
	requireContains(t, "docs/adr/0002-use-redis-fast-path.md", "context", "decision", "consequences")
}

func TestLevel09MessagingArtifacts(t *testing.T) {
	requireContains(t, "migrations/000002_outbox.up.sql", "create table outbox_events", "status", "available_at")
	requireContains(t, "docs/adr/0003-transactional-outbox.md", "at-least-once", "decision", "consequences")
}

func TestLevel12ObservabilityArtifacts(t *testing.T) {
	requireContains(t, "docs/observability.md", "http_requests", "p95", "trace", "consumer lag")
}

func TestLevel13ContainersAndKubernetes(t *testing.T) {
	requireContains(t, "Dockerfile", "from", "user", "entrypoint")
	requireContains(t, "docker-compose.yml", "postgres", "redis", "kafka", "healthcheck")
	requireContains(t, "deployments/helm/fanpulse/Chart.yaml", "apiVersion", "name: fanpulse")
	requireContains(t, "deployments/helm/fanpulse/templates/deployment-api.yaml", "readinessProbe", "livenessProbe", "resources")
	requireContains(t, "deployments/helm/fanpulse/templates/hpa.yaml", "HorizontalPodAutoscaler")
}

func TestLevel14DeliveryAndExperiments(t *testing.T) {
	requireContains(t, ".github/workflows/ci.yml", "go test", "golangci", "docker")
	requireContains(t, "loadtest/reward_claim.js", "idempotency-key", "check", "thresholds")
	requireContains(t, "docs/benchmarks/template.md", "git commit", "environment", "p95", "error rate")
	requireContains(t, "docs/failure-experiments.md", "redis", "postgres", "kafka", "pod")
}
