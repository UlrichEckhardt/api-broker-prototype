package main

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

// Common DB settings.
const (
	AppName                    = "api-broker-prototype" // Name registered at MongoDB for better log readability.
	DBName                     = "test"                 // Name of the DB.
	EventCollectionName        = "events"               // Name of the collection with actual events and payload.
	NotificationCollectionName = "notifications"        // Name of the capped collection with notifications.
)

// Connect to the two collections in the DB
func Connect(host string) (*mongo.Collection, *mongo.Collection, error) {
	opts := options.Client().ApplyURI("mongodb://" + host).SetAppName(AppName).SetConnectTimeout(1 * time.Second)
	err := opts.Validate()
	if err != nil {
		return nil, nil, err
	}

	client, err := mongo.NewClient(opts)
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		return nil, nil, err
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, nil, err
	}

	db := client.Database(DBName)
	events := db.Collection(EventCollectionName)
	notifications := db.Collection(NotificationCollectionName)

	return events, notifications, nil
}
