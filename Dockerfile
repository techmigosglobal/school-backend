FROM golang:1.25.0-alpine AS builder

WORKDIR /app

# Required for cgo build path used by sqlite driver.
RUN apk add --no-cache build-base

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /out/main .

FROM alpine:3.21
WORKDIR /app
RUN addgroup -S app && adduser -S app -G app
COPY --from=builder /out/main /app/main
USER app

EXPOSE 8080
CMD ["./main"]
