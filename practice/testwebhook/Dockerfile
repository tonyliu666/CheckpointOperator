# FROM golang:1.22.3 as builder
# WORKDIR /app
# COPY . .
# RUN go build -o main .
# # copy the main file to the alpine image
# FROM alpine:latest
# WORKDIR /app
# COPY --from=builder /app/main /app/main
# CMD ["./main"]
FROM golang:1.22.3 as builder
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