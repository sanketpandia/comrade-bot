-- Politburo rewrite baseline: tables used by /api/v1, auth, and identity flows.
-- Apply to an empty database only. See README.md and migrations/archive/ for legacy schema.
-- Use: psql -v ON_ERROR_STOP=1 -1 ... < 000_core_schema.sql

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE public.va_role AS ENUM (
    'proletariat',
    'bourgeoisie',
    'administrator'
);

CREATE TABLE public.api_keys (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    status boolean DEFAULT false NOT NULL,
    CONSTRAINT api_keys_pkey PRIMARY KEY (id)
);

CREATE TABLE public.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    discord_id character varying(32) NOT NULL,
    if_community_id character varying(30),
    if_api_id uuid,
    is_active boolean DEFAULT false,
    username text,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT users_discord_id_key UNIQUE (discord_id),
    CONSTRAINT users_if_community_id_key UNIQUE (if_community_id)
);

CREATE INDEX idx_users_username ON public.users USING btree (username);

CREATE TABLE public.virtual_airlines (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    code character varying(30) NOT NULL,
    is_active boolean DEFAULT true,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    discord_server_id character varying(32),
    CONSTRAINT virtual_airlines_pkey PRIMARY KEY (id),
    CONSTRAINT virtual_airlines_code_key UNIQUE (code),
    CONSTRAINT virtual_airlines_discord_server_id_key UNIQUE (discord_server_id)
);

CREATE TABLE public.va_user_roles (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    va_id uuid NOT NULL,
    role public.va_role NOT NULL,
    is_active boolean DEFAULT true,
    joined_at timestamp without time zone DEFAULT now(),
    airtable_pilot_id character varying(20),
    callsign character varying(20),
    updated_at timestamp with time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    CONSTRAINT va_user_roles_pkey PRIMARY KEY (id),
    CONSTRAINT va_user_roles_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id),
    CONSTRAINT va_user_roles_va_id_fkey FOREIGN KEY (va_id) REFERENCES public.virtual_airlines(id)
);

CREATE INDEX idx_va_user_roles_user_id ON public.va_user_roles USING btree (user_id);
CREATE INDEX idx_va_user_roles_va_id ON public.va_user_roles USING btree (va_id);
CREATE INDEX idx_va_user_roles_user_va ON public.va_user_roles USING btree (user_id, va_id);

CREATE UNIQUE INDEX va_user_roles_user_va_unique
    ON public.va_user_roles (user_id, va_id);

CREATE UNIQUE INDEX va_user_roles_one_active_administrator
    ON public.va_user_roles (va_id)
    WHERE role = 'administrator'::public.va_role AND is_active = true;

CREATE TABLE public.banned_discord_ids (
    discord_id character varying(32) PRIMARY KEY,
    banned_at timestamp without time zone DEFAULT now() NOT NULL,
    banned_by character varying(32),
    reason text
);

CREATE TABLE public.platform_reports (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    kind character varying(50) NOT NULL,
    status character varying(20) DEFAULT 'open' NOT NULL,
    reporter_discord_id character varying(32) NOT NULL,
    claimed_ifc character varying(30),
    note text,
    va_id uuid REFERENCES public.virtual_airlines(id) ON DELETE SET NULL,
    new_discord_server_id character varying(32),
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    resolved_at timestamp without time zone,
    resolved_by character varying(32)
);

CREATE TABLE public.user_deletion_archives (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    deleted_at timestamp without time zone DEFAULT now() NOT NULL,
    deleted_by character varying(32),
    payload jsonb NOT NULL
);
