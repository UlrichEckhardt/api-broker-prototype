#!/bin/sh
# init script for the PostgreSQL DB
#
# This is executed inside the Postgre container after startup of the DB engine.
# It creates the setup necessary to function as event store, consisting of a
# table for the data, a function and a trigger to emit notifications when new
# data is inserted into the table.

set -eu

psql -U postgres --no-readline --echo-all --echo-hidden --expanded <<EOF
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    external_uuid UUID UNIQUE,
    created timestamp NOT NULL,
    causation_id INTEGER NOT NULL,
    class TEXT NOT NULL,
    payload JSONB NOT NULL
);
CREATE FUNCTION emit_notification ()
    RETURNS TRIGGER
    LANGUAGE plpgsql
    AS \$\$
    begin
        PERFORM pg_notify('notification', CAST(NEW.id AS text));
        RETURN NULL;
    end
    \$\$;
CREATE TRIGGER on_insert
    AFTER INSERT ON events
    FOR EACH ROW
    EXECUTE FUNCTION emit_notification();
EOF
