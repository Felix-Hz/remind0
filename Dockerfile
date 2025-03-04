#
# Stage 1: Build the Go binary
#

FROM golang:1.24 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to cache dependencies
COPY go.mod go.sum ./

# Download deps
RUN go mod tidy

COPY . .

RUN go build -o main .

#
# Stage 2: Create a smaller image to run the application
#

FROM ubuntu:22.04

# Set the working directory in the new container
WORKDIR /root/

ENV TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}

# Copy the binary from the builder stage
COPY --from=builder /app/main .

CMD ["./main"]
