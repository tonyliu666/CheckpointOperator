FROM golang:1.22.3 as builder
WORKDIR /app
COPY . .
RUN go build -o main .
CMD ["/app/main"]