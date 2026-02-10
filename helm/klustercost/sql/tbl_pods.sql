CREATE SCHEMA IF NOT EXISTS klustercost;

CREATE TABLE IF NOT EXISTS klustercost.tbl_pods
(
    idx integer NOT NULL DEFAULT nextval('tbl_pods_idx_seq'::regclass),
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
)

CREATE TABLE IF NOT EXISTS klustercost.tbl_pod_data
(
    idx integer NOT NULL DEFAULT nextval('tbl_pod_data_idx_seq'::regclass),
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
)

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
AS $$
BEGIN
    DECLARE
        pod_exists INTEGER;
        delay CHAR(128);
        compare TIMESTAMP;
    BEGIN
        SELECT arg_shard || ' seconds' INTO delay;
        SELECT arg_time::TIMESTAMP - delay::INTERVAL INTO compare;
        SELECT COUNT(*) INTO pod_exists FROM klustercost.tbl_pods WHERE pod = arg_pod AND time > compare;
        IF pod_exists = 0 THEN
            INSERT INTO klustercost.tbl_pods (time, namespace, app, service, pod, node, cpu, mem, shard)
            VALUES (arg_time, arg_namespace, arg_app, arg_service, arg_pod, arg_node, arg_cpu, arg_mem, arg_shard);
        END IF;
    END;
END;
$$;
