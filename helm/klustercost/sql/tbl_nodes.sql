CREATE SCHEMA IF NOT EXISTS klustercost;
CREATE TABLE klustercost.tbl_nodes (
    idx serial PRIMARY KEY,
    node VARCHAR (100),
    mem VARCHAR (100),
    cpu VARCHAR (100),
    price_per_hour VARCHAR (100)
);


CREATE OR REPLACE PROCEDURE add_node(
    arg_node VARCHAR(100),
    arg_mem VARCHAR(100),
    arg_cpu VARCHAR(100)
)
language plpgsql
as $$
declare
  node_exists INTEGER;
begin
  SELECT COUNT(*) INTO node_exists FROM klustercost.tbl_nodes WHERE node = arg_node;
  IF node_exists = 0 THEN
    INSERT INTO klustercost.tbl_nodes (node, mem, cpu)
    VALUES (arg_node, arg_mem, arg_cpu);
  END IF;
-- stored procedure body
end; $$