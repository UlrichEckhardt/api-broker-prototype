# API Broker Prototype

## Prototype for an event-driven API broker

![Go](https://github.com/UlrichEckhardt/api-broker-prototype/workflows/Go/badge.svg)

This is a prototype for a pattern to access APIs.

It uses ideas prevalent in Domain Driven Design, namely Event Sourcing and CQRS, in order to access a remote API.

### Goals

The particular goal are as follows:

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

### Non-Goals

The following points are explicitly not goals:

- Implementing a full DDD business model.
  The scope (domain boundary) of this design pattern is the API, not the business around it.
- Serving any purpose other than being a proof of concept.
  This project doesn't do anything useful except being an example.

## How to use

### Running things using docker compose

- Choose the storage backend, MongoDB or PostgreSQL. To keep it simple, just put
  `COMPOSE_PROFILES=mongodb-storage` in the .env file for now.
- Start the setup using `docker compose up --build --detach`.
- Configure the broker using
  `docker compose run mongodb-broker configure --retries 2 --timeout 3`.
- Start the event processor using `docker compose run mongodb-broker process`.
  You can adjust the simulated behaviour of the API using various parameters there.
  This will keep running in the foreground.
- In order to simulate a call, `docker compose run mongodb-broker insert request "test"`.
- You should now be able to see the event processor simulating an API call,
  storing the resulting info in the event store.
- Using `docker compose run mongodb-broker list`, you can get a list of all events and
  their contents.
- Finally, you can kill the event processor and shut down the whole setup
  using `docker compose down`.
- In order to purge the collected events, too, run `docker compose down --volumes`.

### Switching the storage backend

To `docker compose`, switching is configured using different profiles.
Instead of putting it into the `.env` file, you can also invoke `docker compose`
with a `--profile` flag on every invocation.

There are two storage backends implemented, one based on MongoDB (the default),
the other on PostgreSQL. In their use, they shouldn't be different. You can
switch using

- Commandline flag `--eventstore-driver postgresql|mongodb`
- Environment variable `EVENTSTORE_DRIVER=postgresql|mongodb`
  In the context of the docker compose setup, you will also have to select the
  correct host for the DB, using
- Commandline flag `--eventstore-db-host postgresql|mongodb`
- Environment variable `EVENTSTORE_DB_HOST=postgresql|mongodb`
  Since the setup uses the "host" network, you can also use "localhost", which
  is also the default.

## Diagnostics

You can use

- Commandline flag `--eventstore-loglevel debug|info|warn|error|crit`
- Environment variable `EVENTSTORE_LOGLEVEL=debug|info|warn|error|crit`
  to control the amount of info logged. Currently, the default is "info",
  while all log messages are at level "debug".
