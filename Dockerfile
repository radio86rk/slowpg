FROM golang:1.21 as builder
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build -o exporter cmd/exporter/main.go

FROM scratch
WORKDIR /app
COPY --from=builder /build/exporter .
ENTRYPOINT ["/app/exporter"] 