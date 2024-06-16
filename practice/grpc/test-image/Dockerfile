# # create a new image from the base directory, test-kafka
FROM golang:1.21.11-alpine3.20 as builder
    RUN apk update && \
    apk add --no-cache \
        build-base \
        git \
        protobuf-dev \
        protobuf \
        libnl3-dev \
        libcap-dev \
        libaio-dev \
        libseccomp-dev \
        expat-dev \
        pkgconfig \
        protobuf-c-dev \
        libnet-dev \
        gnutls-dev \
        xmlto \
        asciidoc \
        zlib-dev \
        libbsd-dev \
        iproute2 \
        python3 \
        py3-pip \
        gpgme-dev \
        btrfs-progs-dev && \
    rm -rf /var/cache/apk/*

# Set the working directory
WORKDIR /app

# Copy the Go modules manifests
COPY go.mod go.sum ./

# Install Go modules
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN go build -o main .

# Set the entrypoint
CMD ["/app/main"]
