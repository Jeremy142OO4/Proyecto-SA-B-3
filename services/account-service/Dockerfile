FROM golang:1.25-alpine AS compilacion

WORKDIR /aplicacion
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/account-service .

FROM alpine:3.22
RUN addgroup -S aplicacion && adduser -S aplicacion -G aplicacion
USER aplicacion
COPY --from=compilacion /bin/account-service /bin/account-service
EXPOSE 8082
ENTRYPOINT ["/bin/account-service"]
