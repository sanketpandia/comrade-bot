-- Per-VA aircraft and livery name → PIREP filing value dictionaries.

CREATE TABLE public.va_pirep_value_mappings (
    va_id uuid NOT NULL,
    aircraft_mappings jsonb DEFAULT '{}'::jsonb NOT NULL,
    airline_mappings jsonb DEFAULT '{}'::jsonb NOT NULL,
    default_aircraft text,
    default_airline text,
    revision bigint NOT NULL,
    updated_at timestamp with time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    updated_by uuid NOT NULL,
    CONSTRAINT va_pirep_value_mappings_pkey PRIMARY KEY (va_id),
    CONSTRAINT va_pirep_value_mappings_va_id_fkey FOREIGN KEY (va_id)
        REFERENCES public.virtual_airlines (id) ON DELETE CASCADE,
    CONSTRAINT va_pirep_value_mappings_aircraft_mappings_object
        CHECK (jsonb_typeof(aircraft_mappings) = 'object'),
    CONSTRAINT va_pirep_value_mappings_airline_mappings_object
        CHECK (jsonb_typeof(airline_mappings) = 'object'),
    CONSTRAINT va_pirep_value_mappings_revision_positive
        CHECK (revision > 0)
);