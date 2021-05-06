# Prototype for an event-driven API broker

![Go](https://github.com/UlrichEckhardt/api-broker-prototype/workflows/Go/badge.svg)

This is a prototype for a pattern to access APIs.

It uses ideas prevalent in Domain Driven Design, namely Event Sourcing and CQRS, in order to access a remote API.

## Particular goals are:

 - An unreliable API is handled gracefully.
   Things like intermittent failures (e.g. caused by network fabric in between)
   doesn't percolate up the call chain and cause disruptions there. Instead, a
   temporary outage causes a delay, which you can deal with and which is present
   anyway (data in an event sourced system is eventually consistent).
 - Business logic and technical code is separated.
   If you have retries and similar error handling strategies intermingled with
   your application logic, you increase complexity. Separating these issues into
   technical requirements (e.g. when to send a request) from business code (how
   to interpret a response) should help.
 - Developing a software design pattern (in the Gang of Four sense) that can be
   applied in any place a remote API is used. The design should be visible and
   relatively pure within the code in order to serve as example. Parameters
   (options) that can be chosen are visible as such and their behaviour is
   understood.

## What isn't the goal of this:

 - Implementing a full DDD business model.
   The scope (domain boundary) of this design pattern is the API, not the business around it.
 - Serving any purpose other than being a proof of concept.
   This project doesn't do anything useful except being an example.


#  How to use this example prototype:

## Running things locally

 - Start a MongoDB using `bash startup.sh`.
 - Configure the broker using e.g. `./api-broker-prototype configure --retries 2`.
   In addition to configuring the broker, it creates an event in the MongoDB
   collection. This is necessary, because you can't wait on an empty capped
   collection. This is a bug (IMHO) in MongoDB and some people even reported
   it as such.
 - Start the event processor using `./api-broker-prototype process`. You can adjust
   the simulated behaviour of the API using various parameters there.
 - In order to simulate a call, `./api-broker-prototype insert request "test"`.
 - You should now be able to see the event processor simulating an API call, storing
   the resulting info in the event store.
 - Using `./api-broker-prototype list`, you can get a list of all events and their
   contents.
 - Finally, you can kill the event processor and shut down the MongoDB using
   `bash shutdown.sh`.

## Running things using docker-compose

 - Start the setup using `docker-compose up --build --detach`.
 - Configure the broker using e.g. `docker-compose broker configure --retries 2`.
   In addition to configuring the broker, it creates an event in the MongoDB
   collection. This is necessary, because you can't wait on an empty capped
   collection. This is a bug (IMHO) in MongoDB and some people even reported
   it as such.
 - Start the event processor using `docker-compose broker process`. You can
   adjust the simulated behaviour of the API using various parameters there.
   This will keep running in the foreground.
 - In order to simulate a call, `docker-compose broker insert request "test"`.
 - You should now be able to see the event processor simulating an API call,
   storing the resulting info in the event store.
 - Using `./api-broker-prototype list`, you can get a list of all events and
   their contents.
 - Finally, you can kill the event processor and shut down the whole setup
   using `docker-compose rm --stop --force`.
