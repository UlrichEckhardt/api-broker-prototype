package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

func main() {
	defer fmt.Println("Done.")

	opts := options.Client().ApplyURI("mongodb://localhost").SetAppName("api-broker-prototype").SetConnectTimeout(1 * time.Second)
	err := opts.Validate()
	if err != nil {
		fmt.Println(err)
		return
	}

	client, err := mongo.NewClient(opts)
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		fmt.Println(err)
		return
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		fmt.Println(err)
		return
	}
}
