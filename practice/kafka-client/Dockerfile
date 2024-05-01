# create a new image from the base directory, test-kafka
FROM golang:1.21.3 as builder
WORKDIR /app
COPY . .
RUN go build -o main .
CMD ["/app/main"]