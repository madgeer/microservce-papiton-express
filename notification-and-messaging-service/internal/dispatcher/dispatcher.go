package dispatcher

import (
	"context"
	"fmt"

	"papiton/notification-service/internal/model"
)

// ─── INTERFACES ───────────────────────────────────────────────────────────────

// NotificationProvider adalah interface untuk provider pengiriman.
// Baik Email maupun Push harus mengimplementasikan interface ini.
type NotificationProvider interface {
	Send(ctx context.Context, msg model.NotificationMessage) error
}

// NotificationRepository adalah interface untuk menyimpan log ke DB
type NotificationRepository interface {
	SaveLog(ctx context.Context, msg model.NotificationMessage, status bool) error
}

// ─── DISPATCHER ──────────────────────────────────────────────────────────────

// Dispatcher merutekan NotificationMessage ke provider yang tepat
type Dispatcher struct {
	emailProvider NotificationProvider
	pushProvider  NotificationProvider
	repo          NotificationRepository
}

func NewDispatcher(
	emailProvider NotificationProvider,
	pushProvider NotificationProvider,
	repo NotificationRepository,
) *Dispatcher {
	return &Dispatcher{
		emailProvider: emailProvider,
		pushProvider:  pushProvider,
		repo:          repo,
	}
}

// Dispatch mengirim pesan melalui channel yang sesuai
func (d *Dispatcher) Dispatch(ctx context.Context, msg model.NotificationMessage) error {
	var provider NotificationProvider

	switch msg.Channel {
	case model.ChannelEmail:
		provider = d.emailProvider
	case model.ChannelPush:
		provider = d.pushProvider
	default:
		return fmt.Errorf("channel tidak dikenal: %s", msg.Channel)
	}

	err := provider.Send(ctx, msg)
	success := err == nil

	// Simpan log — best effort, jangan sampai gagal log menghentikan proses
	_ = d.repo.SaveLog(ctx, msg, success)

	return err
}
