CREATE SCHEMA IF NOT EXISTS klustercost;
CREATE TABLE klustercost.tbl_nodes (
    idx serial PRIMARY KEY,
    node character varying (100),
    mem double precision,
    cpu double precision,
    labels character varying(500),
    "node.kubernetes.io/instance-type" character varying (100),
    "topology.kubernetes.io/region" character varying (100),
    "topology.kubernetes.io/zone" character varying (100),
    "kubernetes.io/os" character varying (100),
    price_per_hour double precision,
    provider_id text
);

CREATE INDEX IF NOT EXISTS tbl_nodes_node
    ON klustercost.tbl_nodes USING hash
    (node COLLATE pg_catalog."default")
    TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS idx_tbl_nodes_provider_id
    ON klustercost.tbl_nodes (lower(provider_id))
    WHERE provider_id IS NOT NULL;

CREATE OR REPLACE PROCEDURE add_node(
	IN arg_node character varying,
	IN arg_mem double precision,
	IN arg_cpu double precision,
	IN arg_labels character varying,
	IN arg_instance_type character varying,
	IN arg_region character varying,
	IN arg_zone character varying,
	IN arg_os character varying,
	IN arg_provider_id character varying)
LANGUAGE 'plpgsql'
AS $$
begin
  IF EXISTS (SELECT 1 FROM klustercost.tbl_nodes WHERE node = arg_node) THEN
    UPDATE klustercost.tbl_nodes SET
      mem = arg_mem,
      cpu = arg_cpu,
      labels = arg_labels,
      "node.kubernetes.io/instance-type" = NULLIF(arg_instance_type, ''),
      "topology.kubernetes.io/region" = NULLIF(arg_region, ''),
      "topology.kubernetes.io/zone" = NULLIF(arg_zone, ''),
      "kubernetes.io/os" = NULLIF(arg_os, ''),
      provider_id = COALESCE(NULLIF(arg_provider_id, ''), provider_id)
    WHERE node = arg_node;
  ELSE
    INSERT INTO klustercost.tbl_nodes (node, mem, cpu, labels,
      "node.kubernetes.io/instance-type", "topology.kubernetes.io/region",
      "topology.kubernetes.io/zone", "kubernetes.io/os", provider_id)
    VALUES (arg_node, arg_mem, arg_cpu, arg_labels,
      NULLIF(arg_instance_type, ''), NULLIF(arg_region, ''),
      NULLIF(arg_zone, ''), NULLIF(arg_os, ''),
      NULLIF(arg_provider_id, ''));
  END IF;
end;
$$;

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
    price_per_hour / cpu AS cpu_price_per_hour,
    provider_id
   FROM tbl_nodes;
