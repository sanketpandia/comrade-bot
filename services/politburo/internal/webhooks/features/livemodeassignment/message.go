package livemodeassignment

import (
	"fmt"
	"strings"
	"time"

	webhookconfig "infinite-experiment/politburo/internal/webhooks/config"
	"infinite-experiment/politburo/internal/webhooks/discord"
)

const embedColor = 0x38A169

// Event is one successful live schedule self-assignment.
type Event struct {
	IFUserID          string
	PilotUsername     string
	DiscordID         string
	Registration      string
	OriginIcao        string
	DestinationIcao   string
	Callsign          string
	ScheduleID        string
}

func buildEmbed(at time.Time, ev Event) discord.Embed {
	route := fmt.Sprintf("%s-%s",
		strings.ToUpper(strings.TrimSpace(ev.OriginIcao)),
		strings.ToUpper(strings.TrimSpace(ev.DestinationIcao)),
	)
	pilotLabel := strings.TrimSpace(ev.PilotUsername)
	if pilotLabel == "" {
		pilotLabel = "—"
	}
	discordMention := "—"
	if id := strings.TrimSpace(ev.DiscordID); id != "" {
		discordMention = fmt.Sprintf("<@%s>", id)
	}
	fields := []discord.EmbedField{
		{Name: "Pilot (IFC)", Value: pilotLabel, Inline: true},
		{Name: "Infinite Flight user", Value: strings.TrimSpace(ev.IFUserID), Inline: true},
		{Name: "Discord", Value: discordMention, Inline: true},
		{Name: "Registration", Value: strings.ToUpper(strings.TrimSpace(ev.Registration)), Inline: true},
		{Name: "Route", Value: route, Inline: true},
	}
	if callsign := strings.TrimSpace(ev.Callsign); callsign != "" {
		fields = append(fields, discord.EmbedField{Name: "Callsign", Value: callsign, Inline: true})
	}
	if scheduleID := strings.TrimSpace(ev.ScheduleID); scheduleID != "" {
		fields = append(fields, discord.EmbedField{Name: "Schedule", Value: scheduleID, Inline: false})
	}
	return discord.Embed{
		Title:       "Live schedule assigned",
		Description: "A pilot assigned themselves to a scheduled flight.",
		Color:       embedColor,
		Fields:      fields,
		Footer:      &discord.EmbedFooter{Text: webhookconfig.FeatureLiveModeAssignment},
		Timestamp:   at.UTC().Format(time.RFC3339),
	}
}
