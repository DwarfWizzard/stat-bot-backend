FROM golang:alpine AS build

ENV CGO_ENABLED 0

ENV GOOS linux

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o svc ./cmd/stat-bot-back/main.go

FROM debian

WORKDIR /build

COPY --from=build /build/svc /build/svc

RUN apt-get update && apt-get install -y openssh-client

CMD ["./svc"]