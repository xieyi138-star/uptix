FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /uptix ./cmd/uptix

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata curl
COPY --from=builder /uptix /usr/local/bin/uptix
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["uptix"]
CMD ["serve"]
