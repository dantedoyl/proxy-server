CREATE UNLOGGED TABLE req
(
    id   SERIAL PRIMARY KEY NOT NULL,
    url  TEXT,
    meth TEXT,
    body TEXT,
    head TEXT
);