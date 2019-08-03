FROM golang:1.12-alpine as builder
WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY . .

RUN go install ./cmd/httptest

RUN ls /go/bin

FROM alpine:3.10.1
COPY --from=builder /go/bin/httptest /usr/local/bin/httptest
ENTRYPOINT ["/usr/local/bin/httptest"]
