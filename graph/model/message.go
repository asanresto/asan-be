package model

import "time"

type Message struct {
	ID        string  `json:"id"`
	Content   string  `json:"content"`
	SenderID  *string `json:"sender_id,omitempty"`
	CreatedAt *int    `json:"created_at,omitempty"`
}

type SendMessagePayload struct {
	Content   string    `bson:"content"`
	SenderID  string    `bson:"sender_id"`
	RoomID    string    `bson:"room_id"`
	CreatedAt time.Time `bson:"created_at"`
}
