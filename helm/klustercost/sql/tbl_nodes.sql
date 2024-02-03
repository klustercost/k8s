CREATE SCHEMA IF NOT EXISTS klustercost;
CREATE TABLE klustercost.tbl_nodes (
    idx serial PRIMARY KEY,
    node_name VARCHAR (100),
    node_creation_time VARCHAR (100),
    node_mem VARCHAR (100),
    node_cpu VARCHAR (100),
    node_uid VARCHAR (100)
);