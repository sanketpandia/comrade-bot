package va

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetVAPIREPValueMappingsUnavailable(t *testing.T) {
	h := &Handler{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/va/pirep-value-mappings", nil)
	h.GetVAPIREPValueMappings(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", rec.Code)
	}
}