# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app
ARG SERVICE_VERSION=local

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s -X bank-usac/service-notification-audit/internal/telemetry.BuildVersion=${SERVICE_VERSION}" -o /app/service-notification-audit .

# Production stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/service-notification-audit .

EXPOSE 8085

CMD ["./service-notification-audit"]
