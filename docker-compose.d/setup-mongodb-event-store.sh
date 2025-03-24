#!/bin/sh
# init script for the MongoDB
#
# This is executed inside the MongoDB container after startup of the DB engine.
# It creates the collections necessary to function as event store. Note that
# MongoDB also creates collections implicitly, but it won't create a capped
# collection for the notifications, which we absolutely need.

set -eu

mongo --eval "db.createCollection('events')"
mongo --eval 'db.events.createIndex({"external_uuid": 1}, {name: "unique_external_uuid_constraint", unique: true, partialFilterExpression: {external_uuid: { $type: "binData"}}})'
mongo --eval "db.createCollection('notifications', {capped:true, size: 1000000})"
# Create an event in the MongoDB "notifications" collection which is
# necessary, because you can't wait on an empty capped collection. This
# is a known bug, see https://jira.mongodb.org/browse/SERVER-13955.
mongo --eval 'db.notifications.insert({_id: 0})'
