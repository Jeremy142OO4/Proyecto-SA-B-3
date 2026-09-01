FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /api-gateway .

FROM alpine:3.22
RUN addgroup -S -g 10001 bank && adduser -S -D -H -u 10001 -G bank bank
COPY --from=builder /api-gateway /usr/local/bin/api-gateway
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api-gateway"]
