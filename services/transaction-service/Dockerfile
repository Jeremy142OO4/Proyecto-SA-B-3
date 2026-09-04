FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /transaction-service .
FROM alpine:3.22
RUN addgroup -S bank && adduser -S -G bank bank
COPY --from=builder /transaction-service /usr/local/bin/transaction-service
USER bank
EXPOSE 8083
ENTRYPOINT ["/usr/local/bin/transaction-service"]
