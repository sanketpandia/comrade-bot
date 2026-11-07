-- Airport reference data (OurAirports airport-codes dataset).
-- ident is the source key: an ICAO code when one exists, otherwise a local ident.
-- coordinates_lat / coordinates_long are double precision, same numeric shape as live flight positions.

CREATE TABLE public.airports (
    ident text NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    elevation_ft integer,
    continent text NOT NULL,
    iso_country text NOT NULL,
    iso_region text NOT NULL,
    municipality text,
    icao_code text,
    iata_code text,
    coordinates_lat double precision NOT NULL,
    coordinates_long double precision NOT NULL,
    CONSTRAINT airports_pkey PRIMARY KEY (ident),
    CONSTRAINT airports_type_check CHECK (type IN (
        'heliport',
        'small_airport',
        'closed',
        'medium_airport',
        'seaplane_base',
        'large_airport',
        'balloonport'
    )),
    CONSTRAINT airports_coordinates_lat_check CHECK (coordinates_lat >= -90 AND coordinates_lat <= 90),
    CONSTRAINT airports_coordinates_long_check CHECK (coordinates_long >= -180 AND coordinates_long <= 180)
);

CREATE UNIQUE INDEX airports_icao_code_key
    ON public.airports (icao_code)
    WHERE icao_code IS NOT NULL;

CREATE UNIQUE INDEX airports_iata_code_key
    ON public.airports (iata_code)
    WHERE iata_code IS NOT NULL;
