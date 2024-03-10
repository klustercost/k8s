CREATE SCHEMA IF NOT EXISTS klustercost;
CREATE TABLE klustercost.tbl_nodes (
    idx serial PRIMARY KEY,
    node VARCHAR (100),
    mem VARCHAR (100),
    cpu VARCHAR (100),
    price_per_hour VARCHAR (100)
);