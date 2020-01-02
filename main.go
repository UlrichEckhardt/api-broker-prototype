package main

import (
	"context"
	"flag"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

//Envelope .
type Envelope struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	Payload bson.M             `bson:"payload"`
}

func main() {
	defer fmt.Println("Done.")

	startAfter := flag.String("start-after", "", "Start processing after the event with this ID.")
	flag.Parse()
	fmt.Println(*startAfter)
	var lastProcessed *primitive.ObjectID
	if *startAfter != "" {
		lp, err := primitive.ObjectIDFromHex(*startAfter)
		if err != nil {
			fmt.Println(err)
			return
		}
		lastProcessed = &lp
	}
	fmt.Println("starting at", lastProcessed)

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

	events := client.Database("test").Collection("events")

	var doc Envelope
	doc.Payload = bson.M{"answer": 42}
	res, err := events.InsertOne(ctx, doc)
	if err != nil {
		fmt.Println(err)
		return
	}
	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		fmt.Println("no ID returned from insert")
		return
	}
	fmt.Println(id)
}
