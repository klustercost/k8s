CREATE SCHEMA IF NOT EXISTS klustercost;
CREATE TABLE klustercost.tbl_services (
    idx serial PRIMARY KEY,
    service_name VARCHAR (100),
    namespace VARCHAR (100),
    service_insertion_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    own_uid VARCHAR (10000),
    app_label VARCHAR (10000),
    labels VARCHAR (10000),
    selector VARCHAR (10000)
);