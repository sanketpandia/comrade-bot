-- Public /contact form submissions (IFC username + message).

CREATE TABLE public.contact_submissions (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    if_community_id character varying(30) NOT NULL,
    message text NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);

CREATE INDEX contact_submissions_created_at_idx
    ON public.contact_submissions (created_at DESC);
