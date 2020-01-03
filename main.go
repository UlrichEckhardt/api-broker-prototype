package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

//Envelope .
type Envelope struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	Created primitive.DateTime `bson:"created"`
	Payload bson.M             `bson:"payload"`
}

func main() {
	defer fmt.Println("Done.")

	startAfter := flag.String("start-after", "", "Start processing after the event with this ID.")
	flag.Parse()
	fmt.Println(*startAfter)
	var lastProcessed primitive.ObjectID
	if *startAfter != "" {
		lp, err := primitive.ObjectIDFromHex(*startAfter)
		if err != nil {
			fmt.Println(err)
			return
		}
		lastProcessed = lp
	}
	fmt.Println("starting at", lastProcessed)

	events, notifications, err := Connect()

	// insert an additional document
	id, err := insertNewDocument(bson.M{"answer": 42}, events, notifications)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("inserted new document", id.Hex())

	// iterate over existing events
	for {
		envelope, err := getNextDocument(lastProcessed, events, notifications)
		if err != nil {
			fmt.Println(err)
			return
		}
		if envelope == nil {
			break
		}
		fmt.Println("loaded envelope", envelope.ID.Hex())
		fmt.Println("created", envelope.Created.Time().Format(time.RFC3339))
		fmt.Println("payload", envelope.Payload)

		// move to next element
		lastProcessed = envelope.ID
	}
}

func insertNewDocument(payload bson.M, events *mongo.Collection, notifications *mongo.Collection) (primitive.ObjectID, error) {
	var doc Envelope
	doc.Created = primitive.NewDateTimeFromTime(time.Now())
	doc.Payload = payload

	res, err := events.InsertOne(context.Background(), doc)
	if err != nil {
		return primitive.NilObjectID, err
	}
	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, errors.New("no ID returned from insert")
	}

	return id, nil
}

func getNextDocument(lastProcessed primitive.ObjectID, events *mongo.Collection, notifications *mongo.Collection) (*Envelope, error) {
	var filter interface{}
	if lastProcessed.IsZero() {
		filter = bson.M{}
	} else {
		filter = bson.M{"_id": bson.M{"$gt": lastProcessed}}
	}

	res := events.FindOne(nil, filter)
	if res.Err() == mongo.ErrNoDocuments {
		// not an error, there are no more documents left
		return nil, nil
	}
	if res.Err() != nil {
		return nil, res.Err()
	}

	var envelope Envelope
	if err := res.Decode(&envelope); err != nil {
		return nil, err
	}

	return &envelope, nil
}
