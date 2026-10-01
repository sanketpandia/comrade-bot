BEGIN;

ALTER TYPE public.va_role RENAME VALUE 'pilot' TO 'proletariat';
ALTER TYPE public.va_role RENAME VALUE 'staff' TO 'bourgeoisie';
ALTER TYPE public.va_role RENAME VALUE 'admin' TO 'administrator';

ALTER TABLE public.users DROP COLUMN IF EXISTS otp;

CREATE TABLE IF NOT EXISTS public.banned_discord_ids (
    discord_id character varying(32) PRIMARY KEY,
    banned_at timestamp without time zone DEFAULT now() NOT NULL,
    banned_by character varying(32),
    reason text
);

CREATE TABLE IF NOT EXISTS public.platform_reports (
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

CREATE TABLE IF NOT EXISTS public.user_deletion_archives (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    deleted_at timestamp without time zone DEFAULT now() NOT NULL,
    deleted_by character varying(32),
    payload jsonb NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS va_user_roles_user_va_unique
    ON public.va_user_roles (user_id, va_id);

CREATE UNIQUE INDEX IF NOT EXISTS va_user_roles_one_active_administrator
    ON public.va_user_roles (va_id)
    WHERE role = 'administrator'::public.va_role AND is_active = true;

COMMIT;
