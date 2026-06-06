FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/api ./cmd/api && \
    CGO_ENABLED=0 go build -o /app/ingester ./cmd/ingester

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/api .
COPY --from=builder /app/ingester .
COPY migrations/ ./migrations/
EXPOSE 8080
CMD ["./api"]
