FROM golang:1.26-alpine AS builder

ARG SERVICE

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /bin/service ./cmd/${SERVICE}

FROM alpine:3.21

COPY --from=builder /bin/service /bin/service

ENTRYPOINT ["/bin/service"]