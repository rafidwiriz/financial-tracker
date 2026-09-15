package domain

import "time"

type Account struct {
	ID        string `json:"id"`
	UserID    string `json:"-"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Balance   int64  `json:"balance"`
	Currency  string `json:"currency"`
	IsClosed  bool   `json:"is_closed"`
	Color     string `json:"color"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
