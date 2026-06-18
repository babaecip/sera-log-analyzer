FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY agent/go.mod agent/go.sum* ./agent/
COPY agent/*.go ./agent/

WORKDIR /app/agent

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o agent .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/agent/agent .

VOLUME ["/var/log", "/tmp", "/logs"]

CMD ["./agent"]
