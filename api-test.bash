#!/bin/bash

set -e

URL=http://api.docker.localhost

# run `curl` and return a JSON object with the result variables
function json_curl ()
{
    curl --silent -o /dev/null -w '%{json}' "$@" | jq .
}

UUID=$(uuidgen)
echo "generated UUID ${UUID}"

echo "checking API availability with HEAD, should return 200 OK"
RESPONSE=$(json_curl --head ${URL}/up)
echo $RESPONSE | jq --exit-status ".http_code==200" > /dev/null

echo "checking API availability with GET, should return 200 OK"
RESPONSE=$(json_curl ${URL}/up)
echo $RESPONSE | jq --exit-status ".http_code==200" > /dev/null

echo "checking if any such request exists, should return 404 Not Found"
RESPONSE=$(json_curl ${URL}/request/${UUID})
echo $RESPONSE | jq --exit-status ".http_code==404" > /dev/null

echo "checking if creating a request works, should return 201 Created"
RESPONSE=$(json_curl -X POST --data-raw '{"data": "Foo"}' ${URL}/request/${UUID})
echo $RESPONSE | jq --exit-status ".http_code==201" > /dev/null

echo "checking retrieval of the posted event, should return 200 OK"
RESPONSE=$(json_curl ${URL}/request/${UUID})
echo $RESPONSE | jq --exit-status ".http_code==200" > /dev/null

echo "checking that creating an identical request fails, should return 409 Conflict"
RESPONSE=$(json_curl -X POST --data-raw '{"data": "Foo"}' ${URL}/request/${UUID})
echo $RESPONSE | jq --exit-status ".http_code==409" > /dev/null
echo "finished."