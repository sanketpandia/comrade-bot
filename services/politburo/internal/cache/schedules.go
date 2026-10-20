package cache

// Cron schedules for scheduled jobs (robfig/cron with seconds field).
const (
	ScheduleSessionsSync = "0 */5 * * * *"
	ScheduleFlightsSync  = "0 * * * * *"
	ScheduleLiveriesSync = "0 */30 * * * *"
	// ScheduleWebhooksSync fires at minute 0 and 30. Feature frequency gates use that UTC clock.
	ScheduleWebhooksSync = "0 */30 * * * *"
)
