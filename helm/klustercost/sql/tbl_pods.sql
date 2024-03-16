CREATE SCHEMA IF NOT EXISTS klustercost;
CREATE TABLE klustercost.tbl_pods (
    idx serial PRIMARY KEY,
    "time" timestamp without time zone,
    namespace character varying(100),
    app character varying(100),
    service character varying(100),
    pod character varying(100),
    node character varying(10000),
    cpu double precision,
    mem double precision,
    shard integer
);

CREATE OR REPLACE PROCEDURE klustercost.register_pod(
	IN arg_time timestamp without time zone,
	IN arg_namespace character varying,
	IN arg_app character varying,
	IN arg_service character varying,
	IN arg_pod character varying,
	IN arg_node character varying,
	IN arg_cpu double precision,
	IN arg_mem double precision,
	IN arg_shard integer)
LANGUAGE 'plpgsql'
AS $$
BEGIN
    DECLARE
        pod_exists INTEGER;
        delay CHAR(128);
        compare TIMESTAMP;
    BEGIN
        SELECT 600 || ' seconds' INTO delay;
        SELECT arg_time::TIMESTAMP - delay::INTERVAL INTO compare;
        SELECT COUNT(*) INTO pod_exists FROM klustercost.tbl_pods WHERE pod = arg_pod AND time > compare;
        IF pod_exists = 0 THEN
            INSERT INTO klustercost.tbl_pods (time, namespace, app, service, pod, node, cpu, mem, shard)
            VALUES (arg_time, arg_namespace, arg_app, arg_service, arg_pod, arg_node, arg_cpu, arg_mem, arg_shard);
        END IF;
    END;
END;
$$;