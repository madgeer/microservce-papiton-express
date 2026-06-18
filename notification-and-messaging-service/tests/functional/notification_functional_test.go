package functional_test

import (
	"context"
	"testing"

	"papiton/notification-service/internal/dispatcher"
	"papiton/notification-service/internal/model"
	"papiton/notification-service/internal/processor"
	"papiton/notification-service/internal/provider"
	"papiton/notification-service/internal/repository"
)

// TestNotificationFlow_OrderCreated_Integration menguji alur lengkap
// menggunakan implementasi in-memory (tanpa Kafka/DB nyata).
// Jalankan dengan: go test ./tests/functional/... -v
//
// Untuk test dengan Kafka nyata, set environment variable:
//   KAFKA_BROKER=localhost:9092
//   gunakan TestNotificationFlow_WithRealKafka (membutuhkan Docker Compose)
func TestNotificationFlow_OrderCreated_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("melewati functional test (mode short)")
	}

	// ── Inisialisasi komponen nyata (bukan mock) ──────────────────────────────
	proc := processor.NewMessageProcessor()
	emailProv := provider.NewEmailProvider("smtp.papiton.id", 587, "noreply@papiton.id", "", "")
	pushProv := provider.NewPushProvider("", "") // mode simulasi: project_id dan service_account_json kosong
	repo := repository.NewInMemoryNotificationRepository()

	disp := dispatcher.NewDispatcher(emailProv, pushProv, repo)

	// ── Simulasi event masuk ──────────────────────────────────────────────────
	event := model.IncomingEvent{
		EventID:   "func-test-001",
		EventType: model.EventOrderCreated,
		UserID:    "user-functional-001",
		AWB:       "PAPITON-FUNC-001",
	}

	ctx := context.Background()

	// ── Proses event ──────────────────────────────────────────────────────────
	notification, err := proc.Process(event)
	if err != nil {
		t.Fatalf("processor gagal: %v", err)
	}

	// ── Dispatch notifikasi ───────────────────────────────────────────────────
	err = disp.Dispatch(ctx, *notification)
	if err != nil {
		t.Fatalf("dispatcher gagal: %v", err)
	}

	// ── Verifikasi log tersimpan ──────────────────────────────────────────────
	if len(repo.Logs) != 1 {
		t.Errorf("seharusnya ada 1 log, dapat: %d", len(repo.Logs))
	}
	if repo.Logs[0].AWB != "PAPITON-FUNC-001" {
		t.Errorf("AWB log salah: %s", repo.Logs[0].AWB)
	}
	if !repo.Logs[0].Success {
		t.Error("log seharusnya menunjukkan status sukses")
	}

	t.Logf("✅ Functional test berhasil! Log: %+v", repo.Logs[0])
}

func TestNotificationFlow_PackageFailed_UsesEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("melewati functional test (mode short)")
	}

	proc := processor.NewMessageProcessor()
	emailProv := provider.NewEmailProvider("smtp.papiton.id", 587, "noreply@papiton.id", "", "")
	pushProv := provider.NewPushProvider("", "") // mode simulasi: project_id dan service_account_json kosong
	repo := repository.NewInMemoryNotificationRepository()
	disp := dispatcher.NewDispatcher(emailProv, pushProv, repo)

	event := model.IncomingEvent{
		EventID:   "func-test-002",
		EventType: model.EventPackageFailed,
		UserID:    "user-functional-002",
		AWB:       "PAPITON-FUNC-002",
		Metadata:  map[string]interface{}{"reason": "Penerima tidak ada di tempat"},
	}

	notification, err := proc.Process(event)
	if err != nil {
		t.Fatalf("processor gagal: %v", err)
	}

	// Verifikasi channel sebelum dispatch
	if notification.Channel != model.ChannelEmail {
		t.Errorf("PackageFailed harus pakai Email, dapat: %s", notification.Channel)
	}

	err = disp.Dispatch(context.Background(), *notification)
	if err != nil {
		t.Fatalf("dispatcher gagal: %v", err)
	}

	// Verifikasi log
	if len(repo.Logs) != 1 {
		t.Errorf("seharusnya ada 1 log, dapat: %d", len(repo.Logs))
	}
	if repo.Logs[0].Channel != string(model.ChannelEmail) {
		t.Errorf("log channel seharusnya email, dapat: %s", repo.Logs[0].Channel)
	}

	t.Logf("✅ PackageFailed correctly routed ke Email. Log: %+v", repo.Logs[0])
}

func TestNotificationFlow_MultipleEvents_AllLogged(t *testing.T) {
	if testing.Short() {
		t.Skip("melewati functional test (mode short)")
	}

	proc := processor.NewMessageProcessor()
	emailProv := provider.NewEmailProvider("smtp.papiton.id", 587, "noreply@papiton.id", "", "")
	pushProv := provider.NewPushProvider("", "") // mode simulasi: project_id dan service_account_json kosong
	repo := repository.NewInMemoryNotificationRepository()
	disp := dispatcher.NewDispatcher(emailProv, pushProv, repo)
	ctx := context.Background()

	events := []model.IncomingEvent{
		{EventID: "e1", EventType: model.EventOrderCreated, UserID: "u1", AWB: "AWB-001"},
		{EventID: "e2", EventType: model.EventPackagePickedUp, UserID: "u1", AWB: "AWB-001"},
		{EventID: "e3", EventType: model.EventPackageDelivered, UserID: "u1", AWB: "AWB-001"},
	}

	for _, event := range events {
		notif, err := proc.Process(event)
		if err != nil {
			t.Fatalf("gagal proses event %s: %v", event.EventID, err)
		}
		if err := disp.Dispatch(ctx, *notif); err != nil {
			t.Fatalf("gagal dispatch event %s: %v", event.EventID, err)
		}
	}

	// Setiap event harus menghasilkan 1 log
	if len(repo.Logs) != 3 {
		t.Errorf("seharusnya ada 3 log, dapat: %d", len(repo.Logs))
	}

	t.Logf("✅ Semua %d event berhasil diproses dan dicatat", len(repo.Logs))
}
