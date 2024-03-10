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
