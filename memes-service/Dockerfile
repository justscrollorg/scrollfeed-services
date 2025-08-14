# Start from the official Golang image for build
FROM golang:1.21 as builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o memes-service

# Use a minimal base image for running
FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/memes-service /app/memes-service
COPY --from=builder /app/model.go /app/model.go
COPY --from=builder /app/fetcher.go /app/fetcher.go
COPY --from=builder /app/main.go /app/main.go
EXPOSE 8080
CMD ["/app/memes-service"]
