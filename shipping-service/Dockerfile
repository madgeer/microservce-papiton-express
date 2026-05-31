# ==========================================
# STAGE 1: Builder (Untuk Kompilasi Kode)
# ==========================================
FROM golang:1.25-alpine AS builder

# Set working directory di dalam container
WORKDIR /app

# Copy file dependency terlebih dahulu agar Docker bisa melakukan caching
COPY go.mod go.sum ./
RUN go mod download

# Copy seluruh source code
COPY . .

# Build aplikasi menjadi binary statis bernama "shipping-service"
# CGO_ENABLED=0 sangat penting agar binary bisa berjalan di Alpine Linux
RUN CGO_ENABLED=0 GOOS=linux go build -o shipping-service ./cmd/api

# ==========================================
# STAGE 2: Runner (Image Super Ringan)
# ==========================================
FROM alpine:latest

WORKDIR /root/

# Copy HANYA file binary dari STAGE 1
COPY --from=builder /app/shipping-service .

# Ekspose port yang akan digunakan oleh API kita
EXPOSE 8080

# Jalankan aplikasi saat container di-start
CMD ["./shipping-service"]