CREATE SCHEMA IF NOT EXISTS klustercost;
CREATE TABLE klustercost.tbl_workloads (
    id serial PRIMARY KEY,
    rel_name VARCHAR (100),
    name VARCHAR (100),
    namespace VARCHAR (100),
    "group" VARCHAR (100),
    version VARCHAR (100),
    kind VARCHAR (100),
    metadata VARCHAR (100)
);