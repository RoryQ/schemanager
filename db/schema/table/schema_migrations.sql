--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    id character varying(255) NOT NULL,
    checksum character varying(32) DEFAULT ''::character varying NOT NULL,
    execution_time_in_millis integer DEFAULT 0 NOT NULL,
    applied_at timestamp with time zone NOT NULL
);



