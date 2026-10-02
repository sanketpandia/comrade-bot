package metrics

import "github.com/prometheus/client_golang/prometheus"

type Registry struct {
	Prometheus             *prometheus.Registry
	Requests               *prometheus.CounterVec
	RequestDuration        *prometheus.HistogramVec
	CacheOperations        *prometheus.CounterVec
	CacheOperationDuration *prometheus.HistogramVec
	CachePayloadBytes      *prometheus.HistogramVec
	CacheInserts           prometheus.Counter
	JobRuns                *prometheus.CounterVec
	JobDuration            *prometheus.HistogramVec
	JobRunning             *prometheus.GaugeVec
	JobLastSuccess         *prometheus.GaugeVec
	FlightsActive          *prometheus.GaugeVec
	FlightsByPilotState    *prometheus.GaugeVec
	LiveryResolveTotal            *prometheus.CounterVec
	FlightsFilteredTotal          *prometheus.CounterVec
	FlightsComputeTotal           *prometheus.CounterVec
	FlightsFPLSyncScheduledTotal  *prometheus.CounterVec
	FlightsRecordUpdateTotal      *prometheus.CounterVec
	FlightsRecordCacheMissTotal   *prometheus.CounterVec
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
		Help:      "Per-flight compute outcomes during the flights refresh job (fast_path vs full_compute).",
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
		Help:      "Per-flight Redis record write attempts (written, skipped_disabled, skipped_fast_path).",
	}, []string{"server", "result"})
	flightsRecordCacheMissTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "politburo",
		Subsystem: "livegame",
		Name:      "flights_record_cache_miss_total",
		Help:      "Prior enriched flight records missing from cache at compute time.",
	}, []string{"server"})
	registry.MustRegister(
		requests, duration,
		cacheOperations, cacheDuration, cachePayloadBytes, cacheInserts,
		jobRuns, jobDuration, jobRunning, jobLastSuccess,
		flightsActive, flightsByPilotState, liveryResolveTotal, flightsFilteredTotal,
		flightsComputeTotal, flightsFPLSyncScheduledTotal, flightsRecordUpdateTotal, flightsRecordCacheMissTotal,
	)
	return &Registry{
		Prometheus: registry, Requests: requests, RequestDuration: duration,
		CacheOperations: cacheOperations, CacheOperationDuration: cacheDuration,
		CachePayloadBytes: cachePayloadBytes, CacheInserts: cacheInserts,
		JobRuns: jobRuns, JobDuration: jobDuration, JobRunning: jobRunning,
		JobLastSuccess: jobLastSuccess,
		FlightsActive: flightsActive, FlightsByPilotState: flightsByPilotState,
		LiveryResolveTotal: liveryResolveTotal, FlightsFilteredTotal: flightsFilteredTotal,
		FlightsComputeTotal: flightsComputeTotal, FlightsFPLSyncScheduledTotal: flightsFPLSyncScheduledTotal,
		FlightsRecordUpdateTotal: flightsRecordUpdateTotal, FlightsRecordCacheMissTotal: flightsRecordCacheMissTotal,
	}
}
