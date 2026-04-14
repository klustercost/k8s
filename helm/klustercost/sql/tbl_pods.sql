CREATE SCHEMA IF NOT EXISTS klustercost;

CREATE SEQUENCE IF NOT EXISTS klustercost.tbl_pods_idx_seq;
CREATE SEQUENCE IF NOT EXISTS klustercost.tbl_pod_data_idx_seq;

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
    cpu_request double precision DEFAULT NULL,
    cpu_limit double precision DEFAULT NULL,
    mem_request double precision DEFAULT NULL,
    mem_limit double precision DEFAULT NULL,
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
    IN "arg.cpu_request" double precision DEFAULT NULL,
    IN "arg.cpu_limit" double precision DEFAULT NULL,
    IN "arg.mem_request" double precision DEFAULT NULL,
    IN "arg.mem_limit" double precision DEFAULT NULL,
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
			INSERT INTO klustercost.tbl_pods   (name, namespace, node, "app.name", "app.instance", "app.version", "app.component", "app.part-of", "app.managed-by")
			VALUES ("arg.name", "arg.namespace", "arg.node", "arg.app.name", "arg.app.instance", "arg.app.version", "arg.app.component", "arg.app.part-of", "arg.app.managed-by");
			SELECT idx INTO pod_id FROM klustercost.tbl_pods WHERE "arg.name" = klustercost.tbl_pods.name;
		END IF;
	
        SELECT 600 || ' seconds' INTO delay;
        SELECT now() - delay::INTERVAL INTO compare;
        SELECT COUNT(*) INTO sample_count FROM klustercost.tbl_pod_data WHERE idx_pod = pod_id AND timestamp > compare;
        IF sample_count = 0 THEN
            INSERT INTO klustercost.tbl_pod_data (idx_pod, cpu, mem, cpu_request, cpu_limit, mem_request, mem_limit)
            VALUES (pod_id, "arg.cpu", "arg.mem", "arg.cpu_request", "arg.cpu_limit", "arg.mem_request", "arg.mem_limit");
        END IF;	
	END;
$BODY$;

CREATE OR REPLACE VIEW klustercost.tbl_nodes_verbose
 AS
 SELECT idx,
    node,
    mem,
    cpu,
    labels,
    "node.kubernetes.io/instance-type",
    "topology.kubernetes.io/region",
    "topology.kubernetes.io/zone",
    "kubernetes.io/os",
    price_per_hour,
    price_per_hour / mem AS mb_price_per_hour,
    price_per_hour / cpu AS cpu_price_per_hour
   FROM tbl_nodes;

CREATE MATERIALIZED VIEW IF NOT EXISTS klustercost.tbl_pod_data_verbose_mv
TABLESPACE pg_default
AS
 SELECT idx,
    idx_pod,
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
   FROM ( SELECT tbl_pod_data.idx,
            tbl_pod_data.idx_pod,
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
             LEFT JOIN tbl_pods ON tbl_pod_data.idx_pod = tbl_pods.idx
             LEFT JOIN tbl_nodes_verbose ON tbl_pods.node::text = tbl_nodes_verbose.node::text) _
WITH DATA;

CREATE OR REPLACE VIEW klustercost.tbl_pod_data_verbose
 AS
 SELECT idx,
    idx_pod,
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

CREATE INDEX IF NOT EXISTS tbl_pods_idx
    ON klustercost.tbl_pods USING btree
    (idx ASC NULLS LAST)
    WITH (fillfactor=100, deduplicate_items=True)
    TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS tbl_pods_namespace
    ON klustercost.tbl_pods USING hash
    (namespace COLLATE pg_catalog."default")
    TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS tbl_pods_app_name
    ON klustercost.tbl_pods USING hash
    ("app.name" COLLATE pg_catalog."default")
    TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS tbl_pods_app_component
    ON klustercost.tbl_pods USING hash
    ("app.component" COLLATE pg_catalog."default")
    TABLESPACE pg_default;        

CREATE INDEX IF NOT EXISTS tbl_pd_data_idx_pod
    ON klustercost.tbl_pod_data USING btree
    (idx_pod ASC NULLS LAST)
    WITH (fillfactor=100, deduplicate_items=True)
    TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS tbl_pod_data_timestamp
    ON klustercost.tbl_pod_data USING btree
    ("timestamp" ASC NULLS LAST)
    WITH (fillfactor=100, deduplicate_items=True)
    TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS idx_pod_data_verbose_date
    ON klustercost.tbl_pod_data_verbose_mv USING btree
    (date ASC NULLS LAST)
    WITH (fillfactor=100, deduplicate_items=True)
    TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS idx_pod_data_verbose_hour
    ON klustercost.tbl_pod_data_verbose_mv USING btree
    (hour ASC NULLS LAST)
    WITH (fillfactor=100, deduplicate_items=True)
    TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS idx_pod_data_verbose_idx
    ON klustercost.tbl_pod_data_verbose_mv USING btree
    (idx ASC NULLS LAST)
    WITH (fillfactor=100, deduplicate_items=True)
    TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS idx_pod_data_verbose_idx_pod
    ON klustercost.tbl_pod_data_verbose_mv USING btree
    (idx_pod ASC NULLS LAST)
    WITH (fillfactor=100, deduplicate_items=True)
    TABLESPACE pg_default;        
