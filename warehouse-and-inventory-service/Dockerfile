FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build aplikasi menjadi file binary bernama 'warehouse-service'
RUN go build -o warehouse-service ./cmd/main.go

# Pindah ke image kosong yang sangat ringan untuk menjalankan aplikasi
FROM alpine:latest

WORKDIR /app

# Copy file binary dari stage builder sebelumnya
COPY --from=builder /app/warehouse-service .

# Ekspos port yang digunakan oleh aplikasi (sesuai di main.go)
EXPOSE 8080

# Jalankan aplikasi
CMD ["./warehouse-service"]
