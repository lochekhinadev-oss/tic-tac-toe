FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY workers/seed /app/workers/seed
COPY backend/tic-tac-toe /app/backend/tic-tac-toe

WORKDIR /app/workers/seed
RUN go build -o /tic-tac-toe-seed .

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /tic-tac-toe-seed /app/tic-tac-toe-seed

CMD ["/app/tic-tac-toe-seed"]
