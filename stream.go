package kronos

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

type accountKV struct {
	Id             int64   `json:"id"`
	Trader         int64   `json:"trader"`
	Balance        float64 `json:"balance"`
	Currency       string  `json:"crncy"`
	Type           string  `json:"type"`
	MarginSchedul  string  `json:"mrgn_scheduler"`
	MarginLevel    string  `json:"acct_mrgn_lvl"`
	MarginReq      float64 `json:"mrgn_req"`
	MarginCoeff    float64 `json:"mrgn_coeff"`
	MarginDayCoeff float64 `json:"mrgn_day_coeff"`
	OfficeId       int64   `json:"office"`
}

type tradeKV struct {
	Id         int64   `json:"trade"`
	AccountId  int64   `json:"account"`
	PairId     int64   `json:"pair"`
	Amount     float64 `json:"amount"`
	Price      float64 `json:"price"`
	SellBuy    string  `json:"sellbuy"`
	Fee        float64 `json:"fee"`
	Fee2       float64 `json:"fee2"`
	Swap       float64 `json:"swap"`
	Swap2      float64 `json:"swap2"`
	Bonus      float64 `json:"bonus"`
	Opened     string  `json:"opened"`
	Commentary string  `json:"commentary"`
}

func newStream(schema string, url []string) (*nats.Conn, error) {
	options := []nats.Option{nats.Name(schema + ":kronos client")}

	options = append(options, nats.DisconnectHandler(func(nc *nats.Conn) {
		log.Println("NATS Disconnected")
	}))

	options = append(options, nats.ReconnectHandler(func(nc *nats.Conn) {
		log.Println("NATS Reconnect")

	}))

	options = append(options, nats.ClosedHandler(func(nc *nats.Conn) {
		log.Println("NATS closed")
	}))

	connect, err := nats.Connect(strings.Join(url, ","), options...)
	if err != nil {
		return nil, err
	}

	return connect, nil
}

func (s *Service) listenAccounts(loadCh chan<- struct{}) {
	var kv nats.KeyValue
	var count int64

	js, err := s.stream.JetStream()
	if err != nil {
		log.Fatal(err)
	}

	if stream, _ := js.StreamInfo("KV_" + s.schema); stream == nil {
		log.Fatal("bucket KV_" + s.schema + " not found")
	}

	kv, err = js.KeyValue(s.schema)
	if err != nil {
		log.Fatal(err)
	}

	w, err := kv.Watch("account.*")
	if err != nil {
		log.Fatal(err)
	}

	defer w.Stop()

	for kve := range w.Updates() {
		if kve == nil {
			log.Println("accounts loading completed: ", count)

			//release lock
			loadCh <- struct{}{}

			continue
		}

		count++

		acct := &accountKV{}

		err := json.Unmarshal(kve.Value(), acct)
		if err != nil {
			log.Println(err.Error(), string(kve.Value()))
		}

		actual, loaded := s.accounts.LoadOrStore(acct.Id, &Account{
			AccountId:   acct.Id,
			AccountType: acct.Type,
			TraderId:    acct.Trader,
			Balance:     acct.Balance,
			Currency:    acct.Currency,
			OfficeId:    acct.OfficeId,
			Scheduler:   acct.MarginSchedul == "Y",
			Level:       acct.MarginLevel == "Y",
			Require:     acct.MarginReq,
			Coeff:       acct.MarginCoeff,
			DayCoeff:    acct.MarginDayCoeff,
		})

		if loaded {
			account := actual.(*Account)

			account.Balance = acct.Balance
			account.TraderId = acct.Trader
			account.Currency = acct.Currency
		}
	}
}

func (s *Service) listenTrades(loadCh chan<- struct{}) {
	var kv nats.KeyValue
	var count int64

	js, _ := s.stream.JetStream()

	if stream, _ := js.StreamInfo("KV_" + s.schema); stream == nil {
		log.Fatal("bucket KV_" + s.schema + " not found")
	}

	kv, err := js.KeyValue(s.schema)
	if err != nil {
		log.Fatal(err)
	}

	w, err := kv.Watch("trade.*")
	if err != nil {
		log.Fatal(err)
	}

	defer w.Stop()

	for kve := range w.Updates() {
		if kve == nil {
			log.Println("trades loading completed: ", count)

			loadCh <- struct{}{}

			continue
		}

		if kve.Operation() == nats.KeyValuePut {
			count++

			trade := &tradeKV{}

			err := json.Unmarshal(kve.Value(), trade)
			if err != nil {
				log.Println(string(kve.Value()), err.Error())
				continue
			}

			open, err := time.Parse("2006-01-02T15:04:05", trade.Opened)
			if err != nil {
				log.Println(string(kve.Value()), err.Error())
				continue
			}

			s.trades.Store(trade.Id, &Position{
				Id:         trade.Id,
				Account:    trade.AccountId,
				Pair:       trade.PairId,
				Amount:     trade.Amount,
				OpenPrice:  trade.Price,
				Sellbuy:    trade.SellBuy,
				Fee:        trade.Fee,
				Fee2:       trade.Fee2,
				Swap:       trade.Swap,
				Swap2:      trade.Swap2,
				Bonus:      trade.Bonus,
				Opened:     open,
				Commentary: trade.Commentary,
			})

		} else {
			v := strings.Split(kve.Key(), ".")[1]
			id, _ := strconv.Atoi(v)

			s.trades.Delete(int64(id))
		}
	}
}

type Message struct {
	D bool  `json:"d"`
	U []int `json:"u"`
	M struct {
		Event   string `json:"event"`
		Payload struct {
			A int     `json:"a"`
			B float64 `json:"b"`
			E float64 `json:"e"`
			N float64 `json:"n"`
			P float64 `json:"p"`
			U int     `json:"u"`
			C int     `json:"c"`
			O []struct {
				T int     `json:"t"`
				N float64 `json:"n"`
				P float64 `json:"p"`
			} `json:"o"`
		} `json:"payload"`
	} `json:"m"`
}

func (s *Service) listenAccountStates() {
	s.stream.Subscribe(s.schema, func(msg *nats.Msg) {
		m := &Message{}

		err := json.Unmarshal(msg.Data, &m)
		if err != nil {
			return
		}

		if m.M.Event != "account-state" {
			return
		}

		for _, v := range m.M.Payload.O {
			value, load := s.trades.Load(v.T)
			if load {
				trade := value.(*Position)
				trade.Pl = v.P
			}
		}
	})

}
