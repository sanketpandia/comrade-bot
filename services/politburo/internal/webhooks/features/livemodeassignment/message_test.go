package livemodeassignment

import (
	"strings"
	"testing"
	"time"
)

func TestBuildEmbed(t *testing.T) {
	at := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	embed := buildEmbed(at, Event{
		IFUserID:        "if-42",
		PilotUsername:   "CaptainTest",
		DiscordID:       "999",
		Registration:    "ph-znb",
		OriginIcao:      "eham",
		DestinationIcao: "cyyz",
		Callsign:        "TEST01",
		ScheduleID:      "sched-1",
	})
	if !strings.Contains(embed.Description, "assigned") {
		t.Fatalf("description = %q", embed.Description)
	}
	values := map[string]string{}
	for _, f := range embed.Fields {
		values[f.Name] = f.Value
	}
	if values["Infinite Flight user"] != "if-42" || values["Pilot (IFC)"] != "CaptainTest" {
		t.Fatalf("pilot fields = %+v", values)
	}
	if values["Discord"] != "<@999>" {
		t.Fatalf("discord = %q", values["Discord"])
	}
	if values["Registration"] != "PH-ZNB" || values["Route"] != "EHAM-CYYZ" {
		t.Fatalf("flight fields = %+v", values)
	}
	if values["Callsign"] != "TEST01" || values["Schedule"] != "sched-1" {
		t.Fatalf("optional fields = %+v", values)
	}
}
