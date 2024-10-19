FROM golang:1.23.2 as builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

# Use a minimal base image
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main /app/main

# Ensure the binary is executable
RUN chmod +x /app/main

# Run the binary
CMD ["./main"]