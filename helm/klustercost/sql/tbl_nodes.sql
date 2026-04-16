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
    price_per_hour double precision
);


CREATE OR REPLACE PROCEDURE add_node(
	IN arg_node character varying,
	IN arg_mem double precision,
	IN arg_cpu double precision,
	IN arg_labels character varying,
	IN arg_instance_type character varying,
	IN arg_region character varying,
	IN arg_zone character varying,
	IN arg_os character varying)
LANGUAGE 'plpgsql'
AS $$
declare
  node_exists INTEGER;
begin
  SELECT COUNT(*) INTO node_exists FROM klustercost.tbl_nodes WHERE node = arg_node;
  IF node_exists = 0 THEN
    INSERT INTO klustercost.tbl_nodes (node, mem, cpu, labels,
      "node.kubernetes.io/instance-type", "topology.kubernetes.io/region",
      "topology.kubernetes.io/zone", "kubernetes.io/os")
    VALUES (arg_node, arg_mem, arg_cpu, arg_labels,
      arg_instance_type, arg_region, arg_zone, arg_os);
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
    price_per_hour / cpu AS cpu_price_per_hour
   FROM tbl_nodes;
