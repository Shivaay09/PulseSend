FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build tools and SQLite dev libraries for CGO-based sqlite3 driver
RUN apk add --no-cache build-base sqlite-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build for the native architecture (no cross-compilation) so CGO/sqlite works
RUN CGO_ENABLED=1 go build -o server ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates sqlite-libs

WORKDIR /app

COPY --from=builder /app/server /app/server
# Copy email templates needed at runtime
COPY --from=builder /app/templates /app/templates

RUN mkdir -p /data

ENV DATABASE_URL=/data/pulsesend.db
ENV API_PORT=8080
ENV METRICS_PORT=9090
ENV SMTP_HOST=mailpit
ENV SMTP_PORT=1025

EXPOSE 8080 9090

CMD ["/app/server"]
