FROM golang:1.26-alpine AS builder

ARG SERVICE

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /bin/service ./cmd/${SERVICE}

FROM alpine:3.21

COPY --from=builder /bin/service /bin/service

ENTRYPOINT ["/bin/service"]