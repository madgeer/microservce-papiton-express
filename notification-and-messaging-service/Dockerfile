# ==========================================
# STAGE 1: Builder (Untuk Kompilasi Kode)
# ==========================================
FROM golang:1.22-alpine AS builder

# Set working directory di dalam container
WORKDIR /app

# Copy file dependency terlebih dahulu agar Docker bisa melakukan caching
COPY go.mod go.sum ./
RUN go mod download

# Copy seluruh source code
COPY . .

# Build aplikasi menjadi binary statis
RUN CGO_ENABLED=0 GOOS=linux go build -o notification-service ./cmd/main.go

# ==========================================
# STAGE 2: Runner (Image Super Ringan)
# ==========================================
FROM alpine:latest

WORKDIR /root/

# Copy HANYA file binary dari STAGE 1
COPY --from=builder /app/notification-service .

# Ekspose port (opsional, sesuaikan jika ada HTTP server)
EXPOSE 8080

# Jalankan aplikasi saat container di-start
CMD ["./notification-service"]
