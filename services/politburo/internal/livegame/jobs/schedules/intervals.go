// Package schedules contains the cron intervals used by scheduled jobs.
package schedules

import (
	gameflights "infinite-experiment/politburo/internal/livegame/flights"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
)

const (
	// SessionsSync refreshes the active Infinite Flight sessions every five minutes.
	SessionsSync = gamesessions.RefreshSchedule
	// LiveriesSync refreshes aircraft livery names every hour.
	LiveriesSync = gameliveries.RefreshSchedule
	// FlightsSync refreshes active Infinite Flight flights every minute.
	FlightsSync = gameflights.RefreshSchedule
)
