CREATE SCHEMA IF NOT EXISTS klustercost;
CREATE TABLE klustercost.tbl_nodes (
    idx serial PRIMARY KEY,
    node character varying (100),
    mem double precision,
    cpu double precision,
    "node.kubernetes.io/instance-type" character varying (100),
    "topology.kubernetes.io/region" character varying (100),
    "topology.kubernetes.io/zone" character varying (100),
    "kubernetes.io/os" character varying (100),
    price_per_hour double precision
);


CREATE OR REPLACE PROCEDURE add_node(
	IN arg_node character varying,
	IN arg_mem double precision,
	IN arg_cpu double precision)
LANGUAGE 'plpgsql'
AS $$
declare
  node_exists INTEGER;
begin
  SELECT COUNT(*) INTO node_exists FROM klustercost.tbl_nodes WHERE node = arg_node;
  IF node_exists = 0 THEN
    INSERT INTO klustercost.tbl_nodes (node, mem, cpu)
    VALUES (arg_node, arg_mem, arg_cpu);
  END IF;
-- stored procedure body
end;
$$;