# name for the Docker network
BROKER_NETWORK=api-broker-prototype

docker stop mongodb

docker network rm $BROKER_NETWORK


