ARG GO_VERSION=1.22
FROM golang:${GO_VERSION} AS build

RUN mkdir /go/api-broker-prototype
WORKDIR /go/api-broker-prototype

# copy module files separately and download modules
# These only change rarely, so having them in an earlier layer increases the
# chance of Docker cache hits during builds.
COPY go.mod .
COPY go.sum .
RUN go mod download

# copy remaining files and build executable
COPY . .

# build executables statically
RUN CGO_ENABLED=0 go build -C cmd/brittle-api
RUN CGO_ENABLED=0 go build -C cmd/broker


# start a new image, it will only contain the brittle-api executable
FROM scratch AS brittle-api

COPY --from=build /go/api-broker-prototype/cmd/brittle-api/brittle-api /
ENTRYPOINT ["/brittle-api"]


# start a new image, it will only contain the broker executable
FROM scratch AS broker

COPY --from=build /go/api-broker-prototype/cmd/broker/broker /
ENTRYPOINT ["/broker"]
