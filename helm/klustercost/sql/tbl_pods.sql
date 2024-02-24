CREATE SCHEMA IF NOT EXISTS klustercost;
CREATE TABLE klustercost.tbl_pods (
    idx serial PRIMARY KEY,
    pod_name VARCHAR (100),
    namespace VARCHAR (100),
    record_time VARCHAR (100),
    used_mem VARCHAR (100),
    used_cpu VARCHAR (100),
    owner_version VARCHAR (100),
    owner_kind VARCHAR (100),
    owner_name VARCHAR (100),
    owner_uid VARCHAR (100),
    own_uid VARCHAR (100),
    app_label VARCHAR (100),
    labels VARCHAR (10000),
    node_name VARCHAR (100)
);