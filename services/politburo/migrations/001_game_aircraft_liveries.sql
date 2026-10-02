-- Infinite Flight aircraft liveries catalog (synced from Live API bulk /aircraft/liveries).

CREATE TABLE public.game_aircraft_liveries (
    livery_id uuid NOT NULL,
    aircraft_id uuid NOT NULL,
    aircraft_name text NOT NULL,
    livery_name text NOT NULL,
    display_aircraft_name text NOT NULL,
    display_livery_name text NOT NULL,
    created_at timestamp with time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    updated_at timestamp with time zone DEFAULT (now() AT TIME ZONE 'UTC'::text) NOT NULL,
    CONSTRAINT game_aircraft_liveries_pkey PRIMARY KEY (livery_id)
);

CREATE INDEX idx_game_aircraft_liveries_aircraft_id ON public.game_aircraft_liveries USING btree (aircraft_id);
