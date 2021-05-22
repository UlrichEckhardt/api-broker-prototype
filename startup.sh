# read configuration
source config.sh

# image name for the MongoDB server
MONGODB_IMAGE=mongo:4.2

# image name for the Postgres server
POSTGRES_IMAGE=postgres:13.2

docker network create $BROKER_NETWORK

# create MongoDB server
docker run --rm --detach --name mongodb --network $BROKER_NETWORK --publish 27017:27017 $MONGODB_IMAGE
sleep 2 # Hack: wait for the DB to start up
docker exec mongodb mongo --eval "db.createCollection('events')"
docker exec mongodb mongo --eval "db.createCollection('notifications', {capped:true, size: 1000000})"


# create Postgres SQL server
docker run --rm --detach --name postgres --network $BROKER_NETWORK --env POSTGRES_PASSWORD=postgres $POSTGRES_IMAGE
# Hack: wait for the DB to start up
while true; do
    docker exec -it postgres psql -U postgres --command="SELECT 1;" 2>&1 > /dev/null && break
    sleep 2
done
docker exec -i postgres psql -U postgres --no-readline --echo-all --echo-hidden --expanded <<EOF
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    created timestamp NOT NULL,
    causation_id INTEGER,
    class TEXT,
    payload JSONB
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
