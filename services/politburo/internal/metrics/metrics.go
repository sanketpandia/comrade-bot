package metrics

import "github.com/prometheus/client_golang/prometheus"

type Registry struct {
	Prometheus                   *prometheus.Registry
	Requests                     *prometheus.CounterVec
	RequestDuration              *prometheus.HistogramVec
	CacheOperations              *prometheus.CounterVec
	CacheOperationDuration       *prometheus.HistogramVec
	CachePayloadBytes            *prometheus.HistogramVec
	CacheInserts                 prometheus.Counter
	JobRuns                      *prometheus.CounterVec
	JobDuration                  *prometheus.HistogramVec
	JobRunning                   *prometheus.GaugeVec
	JobLastSuccess               *prometheus.GaugeVec
	FlightsActive                *prometheus.GaugeVec
	FlightsByPilotState          *prometheus.GaugeVec
	LiveryResolveTotal           *prometheus.CounterVec
	FlightsFilteredTotal         *prometheus.CounterVec
	FlightsComputeTotal          *prometheus.CounterVec
	FlightsFPLSyncScheduledTotal *prometheus.CounterVec
	FlightsRecordUpdateTotal     *prometheus.CounterVec
	FlightsRecordCacheMissTotal  *prometheus.CounterVec
	FlightsByPhase               *prometheus.GaugeVec
	FlightStateTransitionsTotal  *prometheus.CounterVec
	FlightsAPPlusCandidates      *prometheus.GaugeVec
	FlightsByDataQuality         *prometheus.GaugeVec
	VAActiveFlights              *prometheus.GaugeVec
	WebhookDispatchTotal         *prometheus.CounterVec
	WebhookDispatchDuration      *prometheus.HistogramVec
	WebhookDiscordPosts          *prometheus.CounterVec
	WebhookActiveFlightsServers  *prometheus.HistogramVec
	WebhookActiveFlightsMatched  *prometheus.CounterVec
	WebhookActiveFlightsReported *prometheus.CounterVec
	WebhookActiveFlightsCapped   *prometheus.CounterVec
}

func NewRegistry() *Registry {
	registry := prometheus.NewRegistry()
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Total HTTP requests.",
	}, []string{"method", "route", "status"})
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "politburo",
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "HTTP request duration in seconds.",
	}, []string{"method", "route"})
	cacheOperations := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "cache",
		Name:      "operations_total",
		Help:      "Total cache operations by operation and outcome.",
	}, []string{"operation", "outcome"})
	cacheDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "politburo",
		Subsystem: "cache",
		Name:      "operation_duration_seconds",
		Help:      "Cache operation duration in seconds.",
	}, []string{"operation"})
	cachePayloadBytes := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "politburo",
		Subsystem: "cache",
		Name:      "payload_bytes",
		Help:      "Encoded cache payload size in bytes by operation.",
		Buckets:   prometheus.ExponentialBuckets(128, 4, 9),
	}, []string{"operation"})
	cacheInserts := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "cache",
		Name:      "inserts_total",
		Help:      "Total successful cache inserts.",
	})
	jobRuns := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "jobs",
		Name:      "runs_total",
		Help:      "Total scheduled job runs by job and outcome.",
	}, []string{"job", "outcome"})
	jobDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "politburo",
		Subsystem: "jobs",
		Name:      "run_duration_seconds",
		Help:      "Scheduled job run duration in seconds.",
		Buckets:   prometheus.ExponentialBuckets(0.1, 2, 12),
	}, []string{"job"})
	jobRunning := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "politburo",
		Subsystem: "jobs",
		Name:      "running",
		Help:      "Whether a scheduled job is currently running.",
	}, []string{"job"})
	jobLastSuccess := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "politburo",
		Subsystem: "jobs",
		Name:      "last_success_timestamp_seconds",
		Help:      "Unix timestamp of the last successful scheduled job run.",
	}, []string{"job"})
	flightsActive := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "politburo",
		Subsystem: "livegame",
		Name:      "flights_active",
		Help:      "Active flights in the latest cached snapshot per normalized server name.",
	}, []string{"server"})
	flightsByPilotState := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "politburo",
		Subsystem: "livegame",
		Name:      "flights_by_pilot_state",
		Help:      "Active flights in the latest upstream poll grouped by pilot state.",
	}, []string{"server", "pilot_state"})
	liveryResolveTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "livegame",
		Name:      "livery_resolve_total",
		Help:      "Livery catalog match outcomes while mapping live flights.",
	}, []string{"outcome"})
	flightsFilteredTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "livegame",
		Name:      "flights_filtered_total",
		Help:      "Flights returned after HTTP query filters on active flight endpoints.",
	}, []string{"server", "endpoint"})
	flightsComputeTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "livegame",
		Name:      "flights_compute_total",
		Help:      "Per-flight resolve outcomes during the flights refresh job (skipped vs resolved).",
	}, []string{"server", "outcome"})
	flightsFPLSyncScheduledTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "livegame",
		Name:      "flights_fpl_sync_scheduled_total",
		Help:      "Flights where compute scheduled an FPL sync on this refresh tick.",
	}, []string{"server"})
	flightsRecordUpdateTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "livegame",
		Name:      "flights_record_update_total",
		Help:      "Per-flight Redis record write attempts (written, skipped).",
	}, []string{"server", "result"})
	flightsRecordCacheMissTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "livegame",
		Name:      "flights_record_cache_miss_total",
		Help:      "Prior enriched flight records missing from cache at compute time.",
	}, []string{"server"})
	flightsByPhase := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "politburo",
		Subsystem: "livegame",
		Name:      "flights_by_phase",
		Help:      "Active flights grouped by estimated flight phase.",
	}, []string{"server", "phase"})
	flightStateTransitionsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "livegame",
		Name:      "flight_state_transitions_total",
		Help:      "Committed flight phase transitions during refresh.",
	}, []string{"server", "from", "to"})
	flightsAPPlusCandidates := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "politburo",
		Subsystem: "livegame",
		Name:      "flights_ap_plus_candidates",
		Help:      "Active flights flagged as AP+ resumption candidates.",
	}, []string{"server"})
	flightsByDataQuality := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "politburo",
		Subsystem: "livegame",
		Name:      "flights_by_data_quality",
		Help:      "Active flights grouped by flight-state data quality.",
	}, []string{"server", "quality"})
	vaActiveFlights := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "politburo",
		Subsystem: "livegame",
		Name:      "va_active_flights",
		Help:      "Active flights in the latest VA flight index for each virtual airline.",
	}, []string{"va_id"})
	webhookDispatchTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "webhooks",
		Name:      "dispatch_total",
		Help:      "Webhook feature dispatches that entered the worker pool.",
	}, []string{"webhook_type", "outcome"})
	webhookDispatchDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "politburo",
		Subsystem: "webhooks",
		Name:      "dispatch_duration_seconds",
		Help:      "Webhook feature handler duration in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"webhook_type"})
	webhookDiscordPosts := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "webhooks",
		Name:      "discord_posts_total",
		Help:      "Discord webhook HTTP posts by feature and outcome.",
	}, []string{"webhook_type", "outcome"})
	webhookActiveFlightsServers := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "politburo",
		Subsystem: "webhooks",
		Name:      "active_flights_servers_searched",
		Help:      "Preferred servers online and searched per active_flights dispatch.",
		Buckets:   []float64{0, 1, 2, 3, 4, 5, 8, 12},
	}, []string{"webhook_type"})
	webhookActiveFlightsMatched := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "webhooks",
		Name:      "active_flights_matched_total",
		Help:      "Active flights matched before the webhook report cap.",
	}, []string{"webhook_type"})
	webhookActiveFlightsReported := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "webhooks",
		Name:      "active_flights_reported_total",
		Help:      "Active flights included in a Discord webhook body.",
	}, []string{"webhook_type"})
	webhookActiveFlightsCapped := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "webhooks",
		Name:      "active_flights_capped_total",
		Help:      "active_flights dispatches whose match count exceeded the report cap.",
	}, []string{"webhook_type"})
	registry.MustRegister(
		requests, duration,
		cacheOperations, cacheDuration, cachePayloadBytes, cacheInserts,
		jobRuns, jobDuration, jobRunning, jobLastSuccess,
		flightsActive, flightsByPilotState, liveryResolveTotal, flightsFilteredTotal,
		flightsComputeTotal, flightsFPLSyncScheduledTotal, flightsRecordUpdateTotal, flightsRecordCacheMissTotal,
		flightsByPhase, flightStateTransitionsTotal, flightsAPPlusCandidates, flightsByDataQuality,
		vaActiveFlights,
		webhookDispatchTotal, webhookDispatchDuration, webhookDiscordPosts,
		webhookActiveFlightsServers, webhookActiveFlightsMatched, webhookActiveFlightsReported, webhookActiveFlightsCapped,
	)
	return &Registry{
		Prometheus: registry, Requests: requests, RequestDuration: duration,
		CacheOperations: cacheOperations, CacheOperationDuration: cacheDuration,
		CachePayloadBytes: cachePayloadBytes, CacheInserts: cacheInserts,
		JobRuns: jobRuns, JobDuration: jobDuration, JobRunning: jobRunning,
		JobLastSuccess: jobLastSuccess,
		FlightsActive:  flightsActive, FlightsByPilotState: flightsByPilotState,
		LiveryResolveTotal: liveryResolveTotal, FlightsFilteredTotal: flightsFilteredTotal,
		FlightsComputeTotal: flightsComputeTotal, FlightsFPLSyncScheduledTotal: flightsFPLSyncScheduledTotal,
		FlightsRecordUpdateTotal: flightsRecordUpdateTotal, FlightsRecordCacheMissTotal: flightsRecordCacheMissTotal,
		FlightsByPhase: flightsByPhase, FlightStateTransitionsTotal: flightStateTransitionsTotal,
		FlightsAPPlusCandidates: flightsAPPlusCandidates, FlightsByDataQuality: flightsByDataQuality,
		VAActiveFlights:              vaActiveFlights,
		WebhookDispatchTotal:         webhookDispatchTotal,
		WebhookDispatchDuration:      webhookDispatchDuration,
		WebhookDiscordPosts:          webhookDiscordPosts,
		WebhookActiveFlightsServers:  webhookActiveFlightsServers,
		WebhookActiveFlightsMatched:  webhookActiveFlightsMatched,
		WebhookActiveFlightsReported: webhookActiveFlightsReported,
		WebhookActiveFlightsCapped:   webhookActiveFlightsCapped,
	}
}

func (r *Registry) ObserveWebhookDispatch(webhookType, outcome string, seconds float64) {
	if r == nil {
		return
	}
	r.WebhookDispatchTotal.WithLabelValues(webhookType, outcome).Inc()
	r.WebhookDispatchDuration.WithLabelValues(webhookType).Observe(seconds)
}

func (r *Registry) ObserveWebhookDiscordPost(webhookType, outcome string) {
	if r == nil {
		return
	}
	r.WebhookDiscordPosts.WithLabelValues(webhookType, outcome).Inc()
}

func (r *Registry) ObserveActiveFlightsDispatch(webhookType string, servers, matched, reported int, capped bool) {
	if r == nil {
		return
	}
	r.WebhookActiveFlightsServers.WithLabelValues(webhookType).Observe(float64(servers))
	r.WebhookActiveFlightsMatched.WithLabelValues(webhookType).Add(float64(matched))
	r.WebhookActiveFlightsReported.WithLabelValues(webhookType).Add(float64(reported))
	if capped {
		r.WebhookActiveFlightsCapped.WithLabelValues(webhookType).Inc()
	}
}
