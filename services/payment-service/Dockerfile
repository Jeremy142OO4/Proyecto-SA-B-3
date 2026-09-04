FROM golang:1.25-alpine AS compilacion
WORKDIR /aplicacion
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/payment-service .

FROM alpine:3.22
RUN addgroup -S aplicacion && adduser -S aplicacion -G aplicacion
USER aplicacion
COPY --from=compilacion /bin/payment-service /bin/payment-service
EXPOSE 8084
ENTRYPOINT ["/bin/payment-service"]
