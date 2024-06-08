# Stage 1: Build the Go binary
FROM golang:1.22.4-bullseye as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

# Build the Go binary with static linking
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/main.go

# Stage 2: Create the final image
FROM alpine:3.14.2

WORKDIR /app

# Copy the statically linked binary from the builder stage
COPY --from=builder /app/main .

# Ensure the binary has executable permissions
RUN chmod +x /app/main

CMD ["/app/main"]
