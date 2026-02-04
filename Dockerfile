FROM golang:1.25.6 AS builder

WORKDIR /app

# Optimize: Copy go.mod and go.sum first to leverage Docker cache for dependencies
COPY go.mod go.sum ./

# Download dependencies - this layer will be cached unless go.mod/go.sum changes
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main ./main.go ./user_apis.go

FROM alpine:latest AS runner

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]
