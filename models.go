package kronos

import (
	"sync"
	"time"
)

type Symbol struct {
	ID           int64     `json:"-"`
	Symbol       string    `json:"Symbol"`
	Market       string    `json:"Market"`
	Quote        string    `json:"Quote"`
	Desc         string    `json:"Description"`
	Active       string    `json:"Active"`
	MinTradeSize float64   `json:"MinTradeSize"`
	Precision    float64   `json:"Precision"`
	ContractSize float64   `json:"ContractSize"`
	Group        string    `json:"Group"`
	Type         string    `json:"Type"`
	Bid          float64   `json:"Sell"`
	Ask          float64   `json:"Buy"`
	Date         time.Time `json:"PriceDate"`
}

type Position struct {
	Id         int64
	Account    int64
	Pair       int64
	Amount     float64
	OpenPrice  float64
	Sellbuy    string
	Fee        float64
	Fee2       float64
	Swap       float64
	Swap2      float64
	Bonus      float64
	Pl         float64
	Opened     time.Time
	Commentary string
}

type Account struct {
	AccountId   int64
	AccountType string
	TraderId    int64
	Balance     float64
	Currency    string
	Opentrades  sync.Map
	Equity      float64
	OfficeId    int64
	Scheduler   bool
	Level       bool
	Require     float64
	Coeff       float64
	DayCoeff    float64
}
