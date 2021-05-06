#!/bin/sh
# init script for the MongoDB
#
# This is executed inside the MongoDB container after startup of the DB engine.
# It creates the collections necessary to function as event store. Note that
# MongoDB also creates collections implicitly, but it won't create a capped
# collection for the notifications, which we absolutely need.

set -eu

mongo --eval "db.createCollection('events')"
mongo --eval "db.createCollection('notifications', {capped:true, size: 1000000})"
