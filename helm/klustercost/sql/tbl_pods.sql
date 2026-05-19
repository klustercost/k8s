CREATE SCHEMA IF NOT EXISTS klustercost;

CREATE TABLE IF NOT EXISTS klustercost.tbl_pods
(
    uid character varying(63) COLLATE pg_catalog."default" NOT NULL,
    name character varying(63) COLLATE pg_catalog."default",
    namespace character varying(253) COLLATE pg_catalog."default",
    node character varying(253) COLLATE pg_catalog."default",
    "app.name" character varying(63) COLLATE pg_catalog."default",
    "app.instance" character varying(63) COLLATE pg_catalog."default",
    "app.component" character varying(63) COLLATE pg_catalog."default",
    "app.version" character varying(63) COLLATE pg_catalog."default",
    "app.managed-by" character varying(63) COLLATE pg_catalog."default",
    "app.part-of" character varying(63) COLLATE pg_catalog."default",
    CONSTRAINT tbl_pods_pkey PRIMARY KEY (uid)
);

CREATE INDEX IF NOT EXISTS tbl_pods_app_component
    ON klustercost.tbl_pods USING hash
    ("app.component" COLLATE pg_catalog."default")
    TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS tbl_pods_app_name
    ON klustercost.tbl_pods USING hash
    ("app.name" COLLATE pg_catalog."default")
    TABLESPACE pg_default;    

CREATE INDEX IF NOT EXISTS tbl_pods_namespace
    ON klustercost.tbl_pods USING hash
    (namespace COLLATE pg_catalog."default")
    TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS tbl_pods_uid
    ON klustercost.tbl_pods USING btree
    (uid COLLATE pg_catalog."default" ASC NULLS LAST)
    WITH (fillfactor=100, deduplicate_items=True)
    TABLESPACE pg_default;

create type pod_type as (
  uid text,
  name text,
  namespace text,
  node text,
  "app.name" text,
  "app.instance" text,
  "app.component" text,
  "app.version" text,
  "app.managed-by" text
);

CREATE TABLE IF NOT EXISTS klustercost.tbl_pod_data
(
    "timestamp" timestamp without time zone NOT NULL DEFAULT now(),
    uid character varying(63) COLLATE pg_catalog."default" NOT NULL,
    cpu double precision NOT NULL,
    mem double precision NOT NULL,
    cpu_request double precision,
    cpu_limit double precision,
    mem_request double precision,
    mem_limit double precision,
    CONSTRAINT fk_pod_uid FOREIGN KEY (uid)
        REFERENCES klustercost.tbl_pods (uid) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

CREATE INDEX IF NOT EXISTS tbl_pod_data_timestamp
    ON klustercost.tbl_pod_data USING btree
    ("timestamp" ASC NULLS LAST)
    WITH (fillfactor=100, deduplicate_items=True)
    TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS tbl_pod_data_uid
    ON klustercost.tbl_pod_data USING btree
    (uid COLLATE pg_catalog."default" ASC NULLS LAST)
    WITH (fillfactor=100, deduplicate_items=True)
    TABLESPACE pg_default;

create type pod_data_type as (
  uid text,
  cpu double precision,
  mem double precision,
  cpu_request double precision,	
  cpu_limit double precision,
  mem_request double precision,  
  mem_limit double precision
);

CREATE MATERIALIZED VIEW IF NOT EXISTS klustercost.tbl_pod_data_verbose_mv
TABLESPACE pg_default
AS
 SELECT 
 	uid,
    "timestamp",
    cpu,
    mem,
    cpu_request,
    cpu_limit,
    mem_request,
    mem_limit,
    cpu_price,
    mem_price,
        CASE
            WHEN cpu_price > mem_price THEN cpu_price
            ELSE mem_price
        END AS price,
    "timestamp"::date AS date,
    to_char("timestamp", 'HH24'::text)::integer AS hour
   FROM ( SELECT 
   			tbl_pod_data.uid,
            tbl_pod_data."timestamp",
            tbl_pod_data.cpu,
            tbl_pod_data.mem,
            tbl_pod_data.cpu_request,
            tbl_pod_data.cpu_limit,
            tbl_pod_data.mem_request,
            tbl_pod_data.mem_limit,
            tbl_pod_data.cpu * tbl_nodes_verbose.cpu_price_per_hour AS cpu_price,
            tbl_pod_data.mem * tbl_nodes_verbose.mb_price_per_hour AS mem_price
           FROM tbl_pod_data
             LEFT JOIN tbl_pods ON tbl_pod_data.uid = tbl_pods.uid
             LEFT JOIN tbl_nodes_verbose ON tbl_pods.node::text = tbl_nodes_verbose.node::text) _;

CREATE OR REPLACE VIEW klustercost.tbl_pod_data_verbose
 AS
 SELECT 
 	uid,
    "timestamp",
    cpu,
    mem,
    cpu_request,
    cpu_limit,
    mem_request,
    mem_limit,
    cpu_price,
    mem_price,
    price,
    date,
    hour
   FROM tbl_pod_data_verbose_mv;

CREATE OR REPLACE PROCEDURE klustercost.register_pod_json(
	IN pod_sample jsonb)
LANGUAGE 'plpgsql'
AS $BODY$
	#variable_conflict use_column
	DECLARE	
		required_pod_uid CHAR(128);
	BEGIN
		SELECT uid INTO required_pod_uid FROM tbl_pods WHERE uid = (SELECT (jsonb_populate_record(null::pod_type,pod_sample)).uid);
		IF required_pod_uid IS NULL THEN
			INSERT INTO tbl_pods SELECT * FROM jsonb_populate_record(null::pod_type,pod_sample);
			COMMIT;
		END IF;
		INSERT INTO tbl_pod_data (SELECT now(), (jsonb_populate_record(null::pod_data_type,pod_sample)).*);
	END;
$BODY$;
