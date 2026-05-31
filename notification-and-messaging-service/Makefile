.PHONY: generate-mocks test-unit test-functional test-all lint build clean

## Generate mock files dari interfaces
generate-mocks:
	mockgen -source=internal/dispatcher/dispatcher.go \
	        -destination=mocks/mock_dispatcher.go \
	        -package=mocks

## Unit test SAJA (cepat, tanpa infrastruktur eksternal)
test-unit:
	go test ./internal/... -v -short -race -count=1 \
	    -coverprofile=coverage.out
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Unit test selesai! Lihat coverage.html untuk laporan."

## Functional test (butuh Docker Compose berjalan)
test-functional:
	docker-compose -f ../docker-compose.yml up -d zookeeper kafka notification-db
	@echo "Menunggu Kafka siap..."
	sleep 15
	KAFKA_BROKER=localhost:9092 go test ./tests/functional/... -v -timeout 60s
	docker-compose -f ../docker-compose.yml stop zookeeper kafka notification-db
	docker-compose -f ../docker-compose.yml rm -f zookeeper kafka notification-db
	@echo "✅ Functional test selesai!"

## Jalankan semua test
test-all: test-unit test-functional

## Lint & vet
lint:
	go vet ./...
	@which golangci-lint > /dev/null && golangci-lint run || echo "golangci-lint tidak terinstall, skip."

## Build binary
build:
	go build -o bin/notification-service ./cmd/main.go
	@echo "✅ Build selesai: bin/notification-service"

## Hapus file hasil build
clean:
	rm -rf bin/ coverage.out coverage.html

## Tampilkan bantuan
help:
	@echo ""
	@echo "PAPITON Express - Notification Service"
	@echo "======================================="
	@echo "make generate-mocks   - Generate mock dari interfaces"
	@echo "make test-unit        - Jalankan unit test saja (cepat)"
	@echo "make test-functional  - Jalankan functional test (butuh Docker)"
	@echo "make test-all         - Jalankan semua test"
	@echo "make lint             - Jalankan go vet dan golangci-lint"
	@echo "make build            - Build binary service"
	@echo "make clean            - Hapus file build dan coverage"
	@echo ""
