# Stage 1: Builder
FROM golang:1.22.3-alpine3.19 as builder

# Install GCC
RUN apk update && apk add musl-dev && apk add --no-cache gcc

# Set CGO_ENABLED to enable cgo
ENV CGO_ENABLED=1

# Set the working directory
WORKDIR /krakend-gw

RUN mkdir /logs

# Copy
COPY . ./

# Download dependencies
RUN go mod tidy && go mod vendor

# Build the plugin http-logger
RUN go build -buildmode=plugin -o http-logger.so ./cmd/http-logger/main.go

# Stage 2: Krakend
# FROM krakend/builder:2.6.3
FROM devopsfaith/krakend:2.6.3-watch

WORKDIR /etc/krakend

# Set environment variables
ENV PORT 8080

# Expose ports
EXPOSE 8080 8090

# Copy configuration files
COPY ./config/krakend.json /etc/krakend/

# Copy the plugin from the builder stage
COPY --from=builder /krakend-gw/http-logger.so /etc/krakend/plugins/

# Check the Krakend configuration
RUN FC_ENABLE=1 \
    krakend check -t -d -c "/etc/krakend/krakend.json"

# Define the entry point
ENTRYPOINT FC_ENABLE=1 \
    krakend run -d -c "/etc/krakend/krakend.json" -p $PORT

# CMD ["krakend" "run" "-c" "/etc/krakend/krakend.json"]