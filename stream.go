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
	Id        int64   `json:"trade"`
	AccountId int64   `json:"account"`
	PairId    int64   `json:"pair"`
	Amount    float64 `json:"amount"`
	Price     float64 `json:"price"`
	SellBuy   string  `json:"sellbuy"`
	Fee       float64 `json:"fee"`
	Fee2      float64 `json:"fee2"`
	Swap      float64 `json:"swap"`
	Swap2     float64 `json:"swap2"`
	Bonus     float64 `json:"bonus"`
	Opened    string  `json:"opened"`
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

		actual, loaded := s.accounts.LoadOrStore(acct.Id, &account{
			accountId:   acct.Id,
			accountType: acct.Type,
			traderId:    acct.Trader,
			balance:     acct.Balance,
			currency:    acct.Currency,
			officeId:    acct.OfficeId,
			scheduler:   acct.MarginSchedul == "Y",
			level:       acct.MarginLevel == "Y",
			require:     acct.MarginReq,
			coeff:       acct.MarginCoeff,
			dayCoeff:    acct.MarginDayCoeff,
		})

		if loaded {
			account := actual.(*account)

			account.balance = acct.Balance
			account.traderId = acct.Trader
			account.currency = acct.Currency
		}
	}
}

func (s *Service) listenTrades() {
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
				id:        trade.Id,
				account:   trade.AccountId,
				pair:      trade.PairId,
				amount:    trade.Amount,
				openPrice: trade.Price,
				sellbuy:   trade.SellBuy,
				fee:       trade.Fee,
				fee2:      trade.Fee2,
				swap:      trade.Swap,
				swap2:     trade.Swap2,
				bonus:     trade.Bonus,
				opened:    open,
			})

		} else {
			v := strings.Split(kve.Key(), ".")[1]
			id, _ := strconv.Atoi(v)

			s.trades.Delete(int64(id))
		}
	}
}