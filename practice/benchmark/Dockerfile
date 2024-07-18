# Builder stage
FROM golang:1.22.4 as builder
WORKDIR /app
ADD go.mod go.sum main.go ./
RUN go build -o main .
RUN chmod +x main
ENTRYPOINT [ "/app/main" ]

