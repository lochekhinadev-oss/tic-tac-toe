FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY workers/cleanup /app/workers/cleanup

WORKDIR /app/workers/cleanup
RUN go build -o /tic-tac-toe-cleanup .

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /tic-tac-toe-cleanup /app/tic-tac-toe-cleanup

CMD ["/app/tic-tac-toe-cleanup"]
