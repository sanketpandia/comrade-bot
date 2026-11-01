-- VA-scoped Infinite Flight Live API v3 grants and organisation connections.
-- OAuth secrets stay in Redis; access_expires_at drives the refresh job.

CREATE TABLE public.va_live_grants (
    va_id uuid NOT NULL,
    connected_by_user_id uuid NOT NULL,
    status character varying(20) NOT NULL,
    access_expires_at timestamp with time zone NOT NULL,
    last_refresh_error text,
    created_at timestamp with time zone DEFAULT (now() AT TIME ZONE 'UTC') NOT NULL,
    updated_at timestamp with time zone DEFAULT (now() AT TIME ZONE 'UTC') NOT NULL,
    CONSTRAINT va_live_grants_pkey PRIMARY KEY (va_id),
    CONSTRAINT va_live_grants_status_check CHECK (status IN ('active', 'needs_reconnect'))
);

CREATE INDEX va_live_grants_access_expires_at_idx
    ON public.va_live_grants (access_expires_at)
    WHERE status = 'active';

CREATE TABLE public.va_live_connections (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    va_id uuid NOT NULL,
    live_organization_id uuid NOT NULL,
    organization_name character varying(200) NOT NULL,
    organization_role integer NOT NULL,
    linked_by_user_id uuid NOT NULL,
    linked_at timestamp with time zone DEFAULT (now() AT TIME ZONE 'UTC') NOT NULL,
    CONSTRAINT va_live_connections_pkey PRIMARY KEY (id),
    CONSTRAINT va_live_connections_live_organization_id_key UNIQUE (live_organization_id)
);

CREATE INDEX va_live_connections_va_id_idx ON public.va_live_connections (va_id);

CREATE TABLE public.va_live_connection_history (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    va_id uuid NOT NULL,
    live_organization_id uuid NOT NULL,
    event character varying(30) NOT NULL,
    actor_user_id uuid,
    organization_name character varying(200),
    created_at timestamp with time zone DEFAULT (now() AT TIME ZONE 'UTC') NOT NULL,
    CONSTRAINT va_live_connection_history_pkey PRIMARY KEY (id),
    CONSTRAINT va_live_connection_history_event_check CHECK (
        event IN ('linked', 'unlinked', 'skipped_occupied')
    )
);

CREATE INDEX va_live_connection_history_va_id_created_at_idx
    ON public.va_live_connection_history (va_id, created_at DESC);

CREATE TABLE public.live_api_v3_action_history (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    va_id uuid NOT NULL,
    actor_user_id uuid,
    live_organization_id uuid,
    action_name character varying(100) NOT NULL,
    outcome character varying(20) NOT NULL,
    upstream_status integer,
    error_message text,
    meta jsonb DEFAULT '{}'::jsonb NOT NULL,
    started_at timestamp with time zone NOT NULL,
    finished_at timestamp with time zone NOT NULL,
    duration_ms integer NOT NULL,
    CONSTRAINT live_api_v3_action_history_pkey PRIMARY KEY (id),
    CONSTRAINT live_api_v3_action_history_outcome_check CHECK (
        outcome IN ('success', 'error')
    )
);

CREATE INDEX live_api_v3_action_history_va_id_started_at_idx
    ON public.live_api_v3_action_history (va_id, started_at DESC);
