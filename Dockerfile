# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS build

WORKDIR /app

COPY container_src/go.mod container_src/go.sum* ./
COPY container_src/*.go ./
RUN go mod tidy \
    && CGO_ENABLED=0 GOOS=linux go build -o /agent

FROM alpine:3.20

RUN apk add --no-cache bash coreutils git curl ca-certificates \
    && adduser -D -s /bin/bash dev

COPY --from=build /agent /usr/local/bin/agent
COPY container_src/seed /opt/seed

ENV WORKSPACE_DIR=/workspace \
    HOME=/home/dev \
    PATH=/usr/local/bin:/usr/bin:/bin

EXPOSE 8080

CMD ["/usr/local/bin/agent"]
