package processor

import (
	"fmt"

	"papiton/notification-service/internal/model"
)

// MessageProcessor bertanggung jawab mengubah event menjadi pesan notifikasi
type MessageProcessor struct{}

func NewMessageProcessor() *MessageProcessor {
	return &MessageProcessor{}
}

// Process mengkonversi IncomingEvent menjadi NotificationMessage
func (p *MessageProcessor) Process(event model.IncomingEvent) (*model.NotificationMessage, error) {
	if event.UserID == "" {
		return nil, fmt.Errorf("user_id tidak boleh kosong")
	}
	if event.AWB == "" {
		return nil, fmt.Errorf("AWB tidak boleh kosong")
	}

	subject, body, err := p.buildTemplate(event)
	if err != nil {
		return nil, err
	}

	return &model.NotificationMessage{
		UserID:  event.UserID,
		Channel: p.determineChannel(event),
		Subject: subject,
		Body:    body,
		AWB:     event.AWB,
	}, nil
}

// buildTemplate memilih template pesan berdasarkan jenis event
func (p *MessageProcessor) buildTemplate(event model.IncomingEvent) (subject, body string, err error) {
	switch event.EventType {
	case model.EventOrderCreated:
		subject = "Pesanan Berhasil Dibuat - PAPITON Express"
		body = fmt.Sprintf(
			"Halo! Pesanan kamu dengan nomor resi %s telah berhasil dibuat. "+
				"Estimasi pengambilan dalam 1x24 jam.", event.AWB,
		)
	case model.EventPackagePickedUp:
		subject = "Paket Kamu Sudah Dijemput!"
		body = fmt.Sprintf(
			"Paket dengan resi %s sudah dijemput oleh kurir kami pada %s.",
			event.AWB, event.OccurredAt.Format("02 Jan 2006, 15:04"),
		)
	case model.EventPackageInTransit:
		locationVal, _ := event.Metadata["location"]
		location := fmt.Sprintf("%v", locationVal)
		subject = "Update Perjalanan Paketmu"
		body = fmt.Sprintf(
			"Resi %s kini sedang dalam perjalanan dan berada di: %s.",
			event.AWB, location,
		)
	case model.EventPackageDelivered:
		subject = "Paket Berhasil Diterima!"
		body = fmt.Sprintf(
			"Resi %s telah berhasil diterima. Terima kasih sudah menggunakan PAPITON Express!",
			event.AWB,
		)
	case model.EventPackageFailed:
		reasonVal, _ := event.Metadata["reason"]
		reason := fmt.Sprintf("%v", reasonVal)
		subject = "Pengiriman Gagal - Tindakan Diperlukan"
		body = fmt.Sprintf(
			"Maaf, pengiriman resi %s gagal. Alasan: %s. "+
				"Kurir kami akan mencoba kembali dalam waktu dekat.", event.AWB, reason,
		)
	default:
		return "", "", fmt.Errorf("event type tidak dikenal: %s", event.EventType)
	}
	return subject, body, nil
}

// determineChannel menentukan channel notifikasi berdasarkan jenis event
func (p *MessageProcessor) determineChannel(event model.IncomingEvent) model.Channel {
	if event.EventType == model.EventPackageFailed {
		return model.ChannelEmail
	}
	return model.ChannelPush
}
