package httpapi

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestOpenAPIContractHasCoreRoutes(t *testing.T) {
	data, err := OpenAPIJSON()
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		OpenAPI string                    `json:"openapi"`
		Paths   map[string]map[string]any `json:"paths"`
	}
	if err = json.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	if spec.OpenAPI != "3.1.0" {
		t.Fatalf("expected OpenAPI 3.1, got %q", spec.OpenAPI)
	}
	for _, path := range []string{"/api/v1/auth/register", "/api/v1/courses", "/api/v1/courses/{courseID}/reviews", "/api/v1/courses/{courseID}/posts", "/api/v1/reports"} {
		if len(spec.Paths[path]) == 0 {
			t.Errorf("missing API path %s", path)
		}
	}
}

func TestOpenAPIServedByAPIHandler(t *testing.T) {
	r := httptest.NewRecorder()
	New(nil).ServeHTTP(r, httptest.NewRequest("GET", "/openapi.json", nil))
	if r.Code != 200 {
		t.Fatalf("expected 200, got %d", r.Code)
	}
	if !json.Valid(r.Body.Bytes()) {
		t.Fatal("OpenAPI route returned invalid JSON")
	}
}
