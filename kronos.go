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
	debug    bool
}

func NewService(schema, dbConfig string, natsURL []string, debug bool) (*Service, error) {
	s := &Service{
		cacheTime: time.Now(),
		schema:    schema,
		debug:     debug,
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
		loaded := make(chan struct{})

		go func() {
			s.listenTrades(loaded)
		}()

		<-loaded
	}

	//process account states worker
	{
		go func() {
			s.listenAccountStates()
		}()

	}

	return s, nil
}

func (s *Service) GetPositions(traderFilter int64) (out []*Position) {
	acctList := []int64{}

	if traderFilter != 0 {
		s.accounts.Range(func(k, v interface{}) bool {
			acct := v.(*Account)

			if acct.TraderId == traderFilter {
				acctList = append(acctList, acct.AccountId)
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
				if val == pos.Account {
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

	return
}

func (s *Service) GetClosedPositions(traderFilter int64, accountFilter, start, limit int64, from, till time.Time) (out []*ClosedPosition, err error) {

	acctList := []int64{}

	if traderFilter != 0 {
		s.accounts.Range(func(k, v interface{}) bool {
			acct := v.(*Account)

			if acct.TraderId == traderFilter &&
				(acct.AccountId == accountFilter || accountFilter == 0) {
				acctList = append(acctList, acct.AccountId)
			}

			return true
		})
	} else if accountFilter != 0 {
		acctList = append(acctList, accountFilter)
	}

	if len(acctList) == 0 {
		return s.selectHistory(from, till, start, limit)
	}

	return s.selectHistoryAccount(acctList, from, till, start, limit)
}

func (s *Service) GetAccounts(traderFilter, accountFilter int64) (out []*Account) {
	s.accounts.Range(func(k, v interface{}) bool {
		acct := v.(*Account)

		if acct.TraderId == traderFilter || traderFilter == 0 {
			if acct.AccountId == accountFilter || accountFilter == 0 {
				out = append(out, acct)
			}
		}

		return true
	})

	return
}
