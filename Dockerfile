FROM golang:1.23.1-alpine3.20 AS builder

WORKDIR /app

# Disable CGO and set cross-compilation for arm64
ENV CGO_ENABLED=0 GOOS=linux GOARCH=arm64

# Copy go module files and download dependencies
COPY go.mod go.sum /app/
RUN go mod download

# Copy the rest of the application source code
COPY . .

RUN go build -o /binanceupdater ./cmd/binanceupdater/main.go
RUN go build -o /kucoinupdater ./cmd/kucoinupdater/main.go
RUN go build -o /server ./cmd/server/main.go

# Need trading_pairs.yaml in the container
COPY data /data

FROM alpine:3.17

COPY --from=builder /binanceupdater /binanceupdater
COPY --from=builder /kucoinupdater /kucoinupdater
COPY --from=builder /server /server
COPY --from=builder /data /data

# Ensure binaries are executable
RUN chmod +x /binanceupdater /kucoinupdater /server

EXPOSE 9000

CMD ["/server"]