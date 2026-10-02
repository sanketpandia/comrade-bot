package cache

// Cron schedules for scheduled jobs (robfig/cron with seconds field).
const (
	ScheduleSessionsSync = "0 */5 * * * *"
	ScheduleFlightsSync  = "0 * * * * *"
	ScheduleLiveriesSync = "0 */30 * * * *"
)
