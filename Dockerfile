# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /server ./cmd/server/

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /server /app/server
COPY templates/ /app/templates/
COPY static/ /app/static/
COPY internal/db/migrations/ /app/internal/db/migrations/

RUN mkdir -p /app/data /app/photos

EXPOSE 8080

CMD ["/app/server"]
