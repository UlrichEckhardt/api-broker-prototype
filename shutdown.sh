# read configuration
source config.sh

docker stop mongodb

docker stop postgres

docker network rm $BROKER_NETWORK


