package va

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"infinite-experiment/politburo/internal/community/virtualairlines/featureconfig"
)

type stubFeatureMappings struct {
	enabled bool
}

func (s stubFeatureMappings) Enabled() bool { return s.enabled }
func (s stubFeatureMappings) Get(context.Context, string, string) (featureconfig.FeatureView, error) {
	return featureconfig.FeatureView{}, nil
}
func (s stubFeatureMappings) SaveStats(context.Context, string, string, int, int, featureconfig.StatsDocument) (featureconfig.FeatureView, error) {
	return featureconfig.FeatureView{}, nil
}
func (s stubFeatureMappings) SavePIREP(context.Context, string, string, int, int, featureconfig.PIREPDocument) (featureconfig.FeatureView, error) {
	return featureconfig.FeatureView{}, nil
}
func (s stubFeatureMappings) Delete(context.Context, string, string, int) error { return nil }
func (s stubFeatureMappings) Preview(context.Context, string, string, featureconfig.PreviewInput) (featureconfig.PreviewResult, error) {
	return featureconfig.PreviewResult{}, nil
}

func TestGetVAFeatureConfigDisabled(t *testing.T) {
	h := &Handler{featureMappings: stubFeatureMappings{enabled: false}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/va/feature-configs/stats", nil)
	h.GetVAFeatureConfig(rec, req, featureconfig.FeatureStats)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", rec.Code)
	}
}