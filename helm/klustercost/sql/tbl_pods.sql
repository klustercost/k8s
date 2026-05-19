CREATE SCHEMA IF NOT EXISTS klustercost;

CREATE TABLE IF NOT EXISTS klustercost.tbl_pods
(
    uid character varying(63) COLLATE pg_catalog."default" NOT NULL,
    name character varying(63) COLLATE pg_catalog."default",
    namespace character varying(253) COLLATE pg_catalog."default",
    node character varying(253) COLLATE pg_catalog."default",
    "app.name" character varying(63) COLLATE pg_catalog."default",
    "app.instance" character varying(63) COLLATE pg_catalog."default",
    "app.version" character varying(63) COLLATE pg_catalog."default",
    "app.component" character varying(63) COLLATE pg_catalog."default",
    "app.part-of" character varying(63) COLLATE pg_catalog."default",
    "app.managed-by" character varying(63) COLLATE pg_catalog."default",
    CONSTRAINT tbl_pods_pkey PRIMARY KEY (uid)
);

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

create type pod_data_type as (
  uid text,
  cpu double precision,
  mem double precision,
  cpu_request double precision,	
  cpu_limit double precision,
  mem_request double precision,  
  mem_limit double precision
);

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
