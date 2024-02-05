CREATE SCHEMA IF NOT EXISTS klustercost;
CREATE TABLE klustercost.tbl_owners (
    idx serial PRIMARY KEY,
    name VARCHAR (100),
    namespace VARCHAR (100),
    record_time VARCHAR (100),
    own_version VARCHAR (100),
    own_kind VARCHAR (100),
    own_uid VARCHAR (255),
    owner_version VARCHAR (100),
    owner_kind VARCHAR (100),
    owner_name VARCHAR (100),
    owner_uid VARCHAR (255),
    labels VARCHAR (255)
);