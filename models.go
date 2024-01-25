package kronos

import "time"

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
