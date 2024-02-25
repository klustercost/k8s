## Build
FROM golang:1.21-bullseye as build


WORKDIR /app
COPY ./observer/*.go ./
COPY main.go ./
COPY go.mod ./
COPY go.sum ./
RUN go mod download

ADD pkg ./pkg

RUN go build -o /entrypoint

## Deploy
FROM gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build /entrypoint /entrypoint

USER nonroot:nonroot

ENTRYPOINT ["/entrypoint"]