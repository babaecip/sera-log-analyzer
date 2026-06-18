FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

COPY master/go.mod master/go.sum* ./master/
COPY master/*.go ./master/

WORKDIR /app/master

RUN go mod tidy
RUN CGO_ENABLED=1 GOOS=linux go build -o master .

FROM alpine:3.19

RUN apk add --no-cache sqlite-libs ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/master/master .
RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./master"]
