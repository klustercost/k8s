CREATE SCHEMA IF NOT EXISTS klustercost;
CREATE TABLE klustercost.tbl_pods (
    idx serial PRIMARY KEY,
    time TIMESTAMP WITHOUT TIME ZONE,
    namespace VARCHAR (100),
    app VARCHAR (100),
    service VARCHAR (100),
    pod VARCHAR (100),
    node VARCHAR (10000),
    cpu VARCHAR (100),
    mem VARCHAR (100),
    shard VARCHAR (100)
);

CREATE OR REPLACE PROCEDURE klustercost.register_pod(
    IN arg_time TIMESTAMP WITHOUT TIME ZONE,
    IN arg_namespace VARCHAR(100),
    IN arg_app VARCHAR(100),
    IN arg_service VARCHAR(100),
    IN arg_pod VARCHAR(100),
    IN arg_node VARCHAR(10000),
    IN arg_cpu VARCHAR(100),
    IN arg_mem VARCHAR(100),
    IN arg_shard VARCHAR(100)
)
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