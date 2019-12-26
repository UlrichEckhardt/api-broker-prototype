# read configuration
source config.sh

# image name for the MongoDB server
MONGODB_IMAGE=mongo:4.2

docker network create $BROKER_NETWORK

# create MongoDB server
docker run --rm --detach --name mongodb --network $BROKER_NETWORK $MONGODB_IMAGE
sleep 2 # Hack: wait for the DB to start up
docker exec mongodb mongo --eval "db.createCollection('events')"
docker exec mongodb mongo --eval "db.createCollection('notifications', {capped:true, size: 1000000})"
