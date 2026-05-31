package model

import "time"

// TrackingLog merepresentasikan data log pemindaian tunggal dari titik transit.
// Sesuai untuk skema tabel log di database NoSQL (ScyllaDB/Cassandra/MongoDB).
type TrackingLog struct {
	ResiID       string    `json:"resi_id" bson:"resi_id"`
	LocationCode string    `json:"location_code" bson:"location_code"`
	ActivityCode string    `json:"activity_code" bson:"activity_code"`
	PhotoURL     string    `json:"photo_url,omitempty" bson:"photo_url,omitempty"`
	Timestamp    time.Time `json:"timestamp" bson:"timestamp"`
}

// TrackingHistory merepresentasikan struktur balikan untuk API pelanggan.
// Berisi kumpulan log yang sudah diagregasi berdasarkan ID Resi.
type TrackingHistory struct {
	ResiID  string        `json:"resi_id" bson:"resi_id"`
	History []TrackingLog `json:"history" bson:"history"`
}
