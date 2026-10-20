-- Per-VA typed JSON config (one row per va_id + config_type).
-- va_id and updated_by are not foreign keys; existence is checked in the API.

CREATE TABLE public.va_configs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    va_id uuid NOT NULL,
    config_type character varying(50) NOT NULL,
    config jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    updated_by uuid NOT NULL,
    CONSTRAINT va_configs_pkey PRIMARY KEY (id),
    CONSTRAINT va_configs_va_id_config_type_key UNIQUE (va_id, config_type)
);
