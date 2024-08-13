# use golang alpine image 
FROM golang:1.22.4-alpine3.20 as dev-env
WORKDIR /app 
FROM dev-env as build-env
COPY . /app/
RUN go mod download

RUN CGO_ENABLED=0 go build -o /webhook

FROM alpine:3.10 as runtime

COPY --from=build-env /webhook /usr/local/bin/webhook
RUN chmod +x /usr/local/bin/webhook

ENTRYPOINT ["webhook"]