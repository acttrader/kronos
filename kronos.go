package kronos

import (
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/nats-io/nats.go"
)

const maxChanLength = 1000

type Service struct {
	schema     string
	repository *sqlx.DB
	stream     *nats.Conn

	pairs map[string]*Symbol

	cacheTime time.Time

	trades   sync.Map
	accounts sync.Map
}

type account struct {
	accountId   int64
	accountType string
	traderId    int64
	balance     float64
	currency    string
	opentrades  sync.Map
	equity      float64
	officeId    int64
	scheduler   bool
	level       bool
	require     float64
	coeff       float64
	dayCoeff    float64
}

type Position struct {
	id        int64
	account   int64
	pair      int64
	amount    float64
	openPrice float64
	sellbuy   string
	fee       float64
	fee2      float64
	swap      float64
	swap2     float64
	bonus     float64
	pl        float64
	opened    time.Time
}

func NewService(schema, dbConfig string, natsURL []string) (*Service, error) {
	s := &Service{
		cacheTime: time.Now(),
		schema:    schema,
	}

	nc, err := newStream(schema, natsURL)
	if err != nil {
		return nil, err
	}

	db, err := newRepository(dbConfig)
	if err != nil {
		return nil, err
	}

	s.repository = db
	s.stream = nc

	//process accounts worker
	{
		loaded := make(chan struct{})

		go func() {
			s.listenAccounts(loaded)
		}()

		<-loaded
	}

	//process open trades worker
	{
		go func() {
			s.listenTrades()
		}()
	}

	return s, nil
}

func (s *Service) GetPositions(traderFilter int64) (out []*Position, err error) {
	acctList := []int64{}

	if traderFilter != 0 {
		s.accounts.Range(func(k, v interface{}) bool {
			acct := v.(*account)

			if acct.traderId == traderFilter {
				acctList = append(acctList, acct.accountId)
			}

			return true
		})
	}

	acctListLength := len(acctList)

	s.trades.Range(func(k, v interface{}) bool {
		pos := v.(*Position)

		add := false

		if acctListLength > 0 {
			for _, val := range acctList {
				if val == pos.account {
					add = true
				}
			}
		} else {
			add = true
		}

		if add {
			out = append(out, pos)
		}

		return true
	})

	return out, nil
}