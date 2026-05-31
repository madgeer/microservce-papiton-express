# Menggunakan base image Golang versi Alpine untuk meminimalkan ukuran image.
FROM golang:1.21-alpine AS builder

# Menentukan direktori kerja utama di dalam container.
WORKDIR /app

# Menyalin file dependensi dan mengunduhnya sebelum menyalin seluruh kode.
# Ini memanfaatkan layer caching Docker agar proses build lebih cepat.
COPY go.mod go.sum ./
RUN go mod download

# Menyalin seluruh kode sumber aplikasi ke dalam direktori kerja container.
COPY . .

# Melakukan kompilasi kode Go menjadi file binary eksekusi bernama 'tracking-app'.
RUN go build -o tracking-app ./cmd/main.go

# Menggunakan image Alpine minimalis untuk menjalankan aplikasi hasil kompilasi.
FROM alpine:latest
WORKDIR /app

# Menyalin file binary dari tahap 'builder' ke dalam image final.
COPY --from=builder /app/tracking-app .

# Menentukan perintah utama yang dieksekusi saat container dijalankan.
CMD ["./tracking-app"]