CREATE SCHEMA IF NOT EXISTS klustercost;

CREATE TABLE IF NOT EXISTS klustercost.tbl_pods
(
    idx integer NOT NULL DEFAULT nextval('klustercost.tbl_pods_idx_seq'::regclass),
    name character varying(63) COLLATE pg_catalog."default",
    namespace character varying(253) COLLATE pg_catalog."default",
    node character varying(253) COLLATE pg_catalog."default",
    "app.name" character varying(63) COLLATE pg_catalog."default",
    "app.instance" character varying(63) COLLATE pg_catalog."default",
    "app.version" character varying(63) COLLATE pg_catalog."default",
    "app.component" character varying(63) COLLATE pg_catalog."default",
    "app.part-of" character varying(63) COLLATE pg_catalog."default",
    "app.managed-by" character varying(63) COLLATE pg_catalog."default",
    CONSTRAINT tbl_pods_pkey PRIMARY KEY (idx)
);

CREATE TABLE IF NOT EXISTS klustercost.tbl_pod_data
(
    idx integer NOT NULL DEFAULT nextval('klustercost.tbl_pod_data_idx_seq'::regclass),
    idx_pod integer NOT NULL,
    "timestamp" timestamp without time zone NOT NULL DEFAULT now(),
    cpu double precision NOT NULL,
    mem double precision NOT NULL,
    CONSTRAINT tbl_pod_data_pkey PRIMARY KEY (idx, idx_pod),
    CONSTRAINT fk_pod_idx FOREIGN KEY (idx_pod)
        REFERENCES klustercost.tbl_pods (idx) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID
);

CREATE OR REPLACE PROCEDURE klustercost.register_pod_data(
	IN "arg.name" character varying,
	IN "arg.namespace" character varying,
	IN "arg.node" character varying,
	IN "arg.cpu" double precision,
	IN "arg.mem" double precision,
	IN "arg.app.name" character varying DEFAULT NULL::character varying,
	IN "arg.app.instance" character varying DEFAULT NULL::character varying,
	IN "arg.app.version" character varying DEFAULT NULL::character varying,
	IN "arg.app.component" character varying DEFAULT NULL::character varying,
	IN "arg.app.part-of" character varying DEFAULT NULL::character varying,
	IN "arg.app.managed-by" character varying DEFAULT NULL::character varying)
LANGUAGE 'plpgsql'
AS $BODY$
	#variable_conflict use_column
    DECLARE	
		pod_id INTEGER;
		delay CHAR(128);
        compare timestamp;
		sample_count INTEGER;
	BEGIN
		SELECT idx INTO pod_id FROM klustercost.tbl_pods WHERE "arg.name" = tbl_pods.name;
		IF pod_id IS NULL THEN
			INSERT INTO klustercost.tbl_pods (name, namespace, node, "app.name")
			VALUES ("arg.name", "arg.namespace", "arg.node", "arg.app.name");
			SELECT idx INTO pod_id FROM klustercost.tbl_pods WHERE "arg.name" = klustercost.tbl_pods.name;
		END IF;
	
        SELECT 600 || ' seconds' INTO delay;
        SELECT now() - delay::INTERVAL INTO compare;
        SELECT COUNT(*) INTO sample_count FROM klustercost.tbl_pod_data WHERE idx_pod = pod_id AND timestamp > compare;
        IF sample_count = 0 THEN
            INSERT INTO klustercost.tbl_pod_data (idx_pod, cpu, mem)
            VALUES (pod_id, "arg.cpu", "arg.mem");
        END IF;	
	END;
$BODY$;