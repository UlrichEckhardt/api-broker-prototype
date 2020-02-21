# read configuration
source config.sh

docker stop mongodb

docker network rm $BROKER_NETWORK


