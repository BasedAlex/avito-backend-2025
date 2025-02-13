package db

import "time"

type InfoResponse struct {
	Coins       int         `json:"coins"`
	Inventory   []Item      `json:"inventory"`
	CoinHistory CoinHistory `json:"coinHistory"`
}

type Item struct {
	Type     string `json:"type"`
	Quantity int    `json:"quantity"`
}

type CoinHistory struct {
	Received []Transaction `json:"received"`
	Sent     []Transaction `json:"sent"`
}

type Transaction struct {
	FromUser        string    `json:"fromUser,omitempty"`
	ToUser          string    `json:"toUser,omitempty"`
	Amount          int       `json:"amount"`
	TransactionDate time.Time `json:"transactionDate,omitempty"`
}

type SendCoinRequest struct {
	ToUser string `json:"toUser" binding:"required"`
	Amount int    `json:"amount" binding:"required"`
}
