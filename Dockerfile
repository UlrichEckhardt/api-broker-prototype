ARG GO_VERSION=1.16
FROM golang:${GO_VERSION} AS build

RUN mkdir /go/api-broker-prototype
WORKDIR /go//api-broker-prototype

# copy module files separately and download modules
# These only change rarely, so having them in an earlier layer increases the
# chance of Docker cache hits during builds.
COPY go.mod .
COPY go.sum .
RUN go mod download

# copy remaining files and build executable
COPY . .

# build statically
RUN CGO_ENABLED=0 go build


# start a new image, it will only contain the executable
FROM scratch

COPY --from=build /go/api-broker-prototype/api-broker-prototype /
ENTRYPOINT ["/api-broker-prototype"]
