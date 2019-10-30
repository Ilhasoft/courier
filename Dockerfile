FROM golang:1.11

WORKDIR /app

COPY . .

RUN cd utils/new_channel/main && go build && ./main
RUN go install -v ./cmd/...

EXPOSE 80
ENTRYPOINT ["courier"]
