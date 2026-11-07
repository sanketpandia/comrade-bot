package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListVALiveScheduleEnums(t *testing.T) {
	handler := &ConnectionsHandler{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/va/live/schedule-enums", nil)
	rec := httptest.NewRecorder()
	handler.ListVALiveScheduleEnums(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Data struct {
			Status     []string `json:"status"`
			FlightType []string `json:"flightType"`
			CrewRole   []string `json:"crewRole"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data.Status) == 0 || body.Data.Status[0] != "Unknown" {
		t.Fatalf("status = %+v", body.Data.Status)
	}
	if len(body.Data.FlightType) == 0 || body.Data.FlightType[1] != "Commercial" {
		t.Fatalf("flightType = %+v", body.Data.FlightType)
	}
	if len(body.Data.CrewRole) != 2 || body.Data.CrewRole[1] != "FirstOfficer" {
		t.Fatalf("crewRole = %+v", body.Data.CrewRole)
	}
}
