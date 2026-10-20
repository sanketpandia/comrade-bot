package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVAConfigQueryAliasMapsLegacyType(t *testing.T) {
	var seen string
	handler := VAConfigQueryAlias()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Query().Get("configType")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/va/config?type=BASIC", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if seen != "BASIC" {
		t.Fatalf("configType = %q, want BASIC", seen)
	}
}
