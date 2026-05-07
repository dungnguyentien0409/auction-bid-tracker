package domain

import "time"

type Bid struct {
	ID        string    `json:"id"`
	ItemID    string    `json:"item_id"`
	UserID    string    `json:"user_id"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}
