-- Append-only audit of each VA webhook feature dispatch that entered the worker pool.
-- Stores outcome and timing only. Discord webhook URLs, raw config, and feature
-- payloads are intentionally absent. No purge job in this phase; the table grows
-- with each dispatched run.

CREATE TABLE public.va_webhook_runs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    va_id uuid NOT NULL,
    webhook_type character varying(50) NOT NULL,
    outcome character varying(20) NOT NULL,
    skip_reason character varying(100),
    discord_messages integer DEFAULT 0 NOT NULL,
    error_message text,
    started_at timestamp with time zone NOT NULL,
    finished_at timestamp with time zone NOT NULL,
    duration_ms integer NOT NULL,
    CONSTRAINT va_webhook_runs_pkey PRIMARY KEY (id),
    CONSTRAINT va_webhook_runs_outcome_check CHECK (outcome IN ('success', 'skipped', 'error'))
);

CREATE INDEX va_webhook_runs_va_id_started_at_idx
    ON public.va_webhook_runs (va_id, started_at DESC);

CREATE INDEX va_webhook_runs_webhook_type_started_at_idx
    ON public.va_webhook_runs (webhook_type, started_at DESC);
