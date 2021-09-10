FROM golang:alpine AS builder

RUN set -ex && \
    apk add --no-cache gcc musl-dev

# Set necessary environmet variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64 \
    SPINUP_PROJECT_DIR=/tmp/spinuplocal \
    ARCHITECTURE=amd64 \
    CF_AUTHORIZATION_TOKEN=replaceme \
    CF_ZONE_ID=replaceme \
    CLIENT_ID=replaceme \
    CLIENT_SECRET=replaceme 

# Move to working directory /build
WORKDIR /build

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY . .

# Build the application
RUN go build -o main .

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from build to main folder
RUN cp /build/main .

# Build a small image
FROM docker/compose

COPY --from=builder /dist/main /
COPY docker-compose-template.yml .
RUN mkdir /tmp/spinuplocal

COPY app.rsa /tmp/spinuplocal
COPY app.rsa.pub /tmp/spinuplocal

# Set necessary environmet variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64 \
    SPINUP_PROJECT_DIR=/tmp/spinuplocal \
    ARCHITECTURE=amd64 \
    CF_AUTHORIZATION_TOKEN=replaceme \
    CF_ZONE_ID=replaceme \
    CLIENT_ID=replaceme \
    CLIENT_SECRET=replaceme 

EXPOSE 4434

# Command to run
ENTRYPOINT ["/main"]

