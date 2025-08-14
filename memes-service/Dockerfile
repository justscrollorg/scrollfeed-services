# Start from the official Golang image for build
FROM golang:1.21 as builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o memes-service

# Use a minimal base image for running
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/memes-service /app/memes-service
EXPOSE 8080
CMD ["/app/memes-service"]
