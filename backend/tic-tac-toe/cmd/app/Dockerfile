FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY backend/tic-tac-toe /app/src/tic-tac-toe

WORKDIR /app/src/tic-tac-toe
RUN go build -o /tic-tac-toe-app ./cmd/app

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /tic-tac-toe-app /app/tic-tac-toe-app

EXPOSE 8080

CMD ["/app/tic-tac-toe-app"]
