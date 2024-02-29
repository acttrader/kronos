package kronos

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

type Symbol struct {
	ID           int64           `json:"-"`
	Symbol       string          `json:"Symbol"`
	Market       string          `json:"Market"`
	Quote        string          `json:"Quote"`
	Desc         string          `json:"Description"`
	Active       string          `json:"Active"`
	MinTradeSize decimal.Decimal `json:"MinTradeSize"`
	Precision    decimal.Decimal `json:"Precision"`
	ContractSize decimal.Decimal `json:"ContractSize"`
	Group        string          `json:"Group"`
	Type         string          `json:"Type"`
	Bid          decimal.Decimal `json:"Sell"`
	Ask          decimal.Decimal `json:"Buy"`
	Date         time.Time       `json:"PriceDate"`
}

type Position struct {
	Id         int64
	Account    int64
	Pair       int64
	Amount     decimal.Decimal
	OpenPrice  decimal.Decimal
	Sellbuy    string
	Fee        decimal.Decimal
	Fee2       decimal.Decimal
	Swap       decimal.Decimal
	Swap2      decimal.Decimal
	Bonus      decimal.Decimal
	Pl         decimal.Decimal
	Opened     time.Time
	Commentary string
	PositionId int64
}

type Account struct {
	AccountId   int64
	AccountType string
	TraderId    int64
	Balance     decimal.Decimal
	Currency    string
	Opentrades  sync.Map
	Equity      decimal.Decimal
	OfficeId    int64
	Scheduler   bool
	Level       bool
	Require     decimal.Decimal
	Coeff       decimal.Decimal
	DayCoeff    decimal.Decimal
}

type ClosedPosition struct {
	Id           int64
	PrntId       *int64
	Account      int64
	Pair         int64
	ContractSize decimal.Decimal
	OpenLots     decimal.Decimal
	CloseLots    decimal.Decimal
	OpenPrice    decimal.Decimal
	ClosePrice   decimal.Decimal
	Sellbuy      string
	Fee          decimal.Decimal
	Fee2         decimal.Decimal
	Swap         decimal.Decimal
	Swap2        decimal.Decimal
	Pl           decimal.Decimal
	Opened       time.Time
	Closed       time.Time
	Commentary   string
	PositionId   int64
}
