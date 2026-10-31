-- Stage 0 初始表结构

CREATE TABLE observations (
    id          INTEGER PRIMARY KEY,
    ip          TEXT NOT NULL,
    source      TEXT NOT NULL,
    fields      TEXT NOT NULL,
    fetched_at  INTEGER NOT NULL,
    err         TEXT
);
CREATE INDEX idx_obs_ip_time ON observations(ip, fetched_at DESC);

CREATE TABLE queries (
    id          INTEGER PRIMARY KEY,
    ip          TEXT NOT NULL,
    queried_at  INTEGER NOT NULL
);
CREATE INDEX idx_queries_ip ON queries(ip, queried_at DESC);

CREATE TABLE quota_usage (
    source   TEXT NOT NULL,
    day      TEXT NOT NULL,
    used     INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (source, day)
);
