###
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o aios_lb ./cmd/aios_lb/main.go

###
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /app/aios_lb /app/aios_lb

EXPOSE 7035

ENTRYPOINT ["/app/aios_lb"]

CMD ["--config", "/app/config.yml", "--listen", ":7035"]
