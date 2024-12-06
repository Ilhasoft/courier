FROM golang:1.23-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download -x

COPY . ./

RUN GOOS=linux GOARCH=amd64 go build -o /bin/mailroom ./cmd/mailroom/*.go

FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder bin/mailroom ./

EXPOSE 8090
ENTRYPOINT ["./courier"]
