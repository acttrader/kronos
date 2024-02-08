package kronos

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"github.com/lib/pq"
	_ "github.com/lib/pq" //...
)

func newRepository(path string) (*sqlx.DB, error) {

	db, err := sqlx.Connect("postgres", path)
	if err != nil {
		errM := fmt.Sprintf("[ERROR] - sqlx.Connect('postgres', '%s')", path)
		return nil, errors.Wrap(err, errM)
	}

	// set DB connection parameters
	{
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(1)
		db.SetConnMaxLifetime(time.Duration(1000 * int(time.Millisecond)))
	}

	return db, nil
}

func (s *Service) selectPairs() ([]*Symbol, error) {
	stmt, err := s.repository.Prepare(`select 
	                            id,
							    name, 
							    market, 
						     	base, 
							    contract_size, 
							    precision	 
						       from ` + s.schema + `.pairs
						       order by name`)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := []*Symbol{}

	for rows.Next() {
		var name, market, base sql.NullString
		var contract, precision sql.NullFloat64
		var id sql.NullInt64

		err = rows.Scan(&id, &name, &market, &base, &contract, &precision)
		if err != nil {
			return nil, err
		}

		result = append(result, &Symbol{
			ID:           id.Int64,
			Symbol:       name.String,
			Market:       market.String,
			Quote:        base.String,
			Precision:    precision.Float64,
			ContractSize: contract.Float64,
		})
	}

	if rows.Err() != nil && rows.Err() != io.EOF {
		return nil, rows.Err()
	}

	return result, nil
}

func (s *Service) selectHistory(from, till time.Time, startFrom, limit int64) ([]*ClosedPosition, error) {
	now := time.Now()

	stmt, err := s.repository.Prepare(`select 
									      trade_id, 
										  prnt_trade_id, 
										  acct_id, 
										  pair_id, 
										  lot_size, 
										  open_lots, 
										  close_lots, 
										  sellbuy_ind, 
										  open_date, 
										  close_date, 
										  open_rate, 
										  close_rate, 
										  profit_loss, 
										  swaps, 
										  comm,
										  commentary,
										  position_id
									  from ` + s.schema + `.trade_history th
									  where close_date between $1 and $2
									    and trade_id >= $3
									  order by trade_id
									  limit $4`)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query(from, till, startFrom, limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := []*ClosedPosition{}

	for rows.Next() {
		var sellBuy, commentary sql.NullString
		var lotSize, openLots, closeLots, openPrice, closePrice, profit, comm, swap sql.NullFloat64
		var tradeId, prntTradeId, acctId, pairId, positionId sql.NullInt64
		var opened, closed sql.NullTime

		err = rows.Scan(
			&tradeId,
			&prntTradeId,
			&acctId,
			&pairId,
			&lotSize,
			&openLots,
			&closeLots,
			&sellBuy,
			&opened,
			&closed,
			&openPrice,
			&closePrice,
			&profit,
			&swap,
			&comm,
			&commentary,
			&positionId,
		)
		if err != nil {
			return nil, err
		}

		result = append(result, &ClosedPosition{
			Id:           tradeId.Int64,
			PrntId:       nilInt64(prntTradeId),
			Account:      acctId.Int64,
			Pair:         pairId.Int64,
			ContractSize: lotSize.Float64,
			OpenLots:     openLots.Float64,
			CloseLots:    closeLots.Float64,
			OpenPrice:    openPrice.Float64,
			ClosePrice:   closePrice.Float64,
			Sellbuy:      sellBuy.String,
			Fee:          comm.Float64,
			Swap:         swap.Float64,
			Pl:           profit.Float64,
			Opened:       opened.Time,
			Closed:       closed.Time,
			Commentary:   commentary.String,
			PositionId:   positionId.Int64,
		})
	}

	if rows.Err() != nil && rows.Err() != io.EOF {
		return nil, rows.Err()
	}

	if s.debug {
		log.Printf("[DEBUG] - selectHistory: '%s')", time.Now().Sub(now))
	}

	return result, nil
}

func (s *Service) selectHistoryAccount(acctList []int64, from, till time.Time, startFrom, limit int64) ([]*ClosedPosition, error) {
	now := time.Now()

	stmt, err := s.repository.Prepare(`select 
									      trade_id, 
										  prnt_trade_id, 
										  acct_id, 
										  pair_id, 
										  lot_size, 
										  open_lots, 
										  close_lots, 
										  sellbuy_ind, 
										  open_date, 
										  close_date, 
										  open_rate, 
										  close_rate, 
										  profit_loss, 
										  swaps, 
										  comm,
										  commentary,
										  position_id
									   from ` + s.schema + `.trade_history th
									   where close_date between $1 and $2
									     and acct_id = any($3::int[])
										 and trade_id >= $4
									   order by trade_id
									   limit $5`)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query(from, till, pq.Array(acctList), startFrom, limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	stmt.QueryRow()

	result := []*ClosedPosition{}

	for rows.Next() {
		var sellBuy, commentary sql.NullString
		var lotSize, openLots, closeLots, openPrice, closePrice, profit, comm, swap sql.NullFloat64
		var tradeId, prntTradeId, acctId, pairId, positionId sql.NullInt64
		var opened, closed sql.NullTime

		err = rows.Scan(
			&tradeId,
			&prntTradeId,
			&acctId,
			&pairId,
			&lotSize,
			&openLots,
			&closeLots,
			&sellBuy,
			&opened,
			&closed,
			&openPrice,
			&closePrice,
			&profit,
			&swap,
			&comm,
			&commentary,
			&positionId,
		)
		if err != nil {
			return nil, err
		}

		result = append(result, &ClosedPosition{
			Id:           tradeId.Int64,
			PrntId:       nilInt64(prntTradeId),
			Account:      acctId.Int64,
			Pair:         pairId.Int64,
			ContractSize: lotSize.Float64,
			OpenLots:     openLots.Float64,
			CloseLots:    closeLots.Float64,
			OpenPrice:    openPrice.Float64,
			ClosePrice:   closePrice.Float64,
			Sellbuy:      sellBuy.String,
			Fee:          comm.Float64,
			Swap:         swap.Float64,
			Pl:           profit.Float64,
			Opened:       opened.Time,
			Closed:       closed.Time,
			Commentary:   commentary.String,
			PositionId:   positionId.Int64,
		})
	}

	if rows.Err() != nil && rows.Err() != io.EOF {
		return nil, rows.Err()
	}

	if s.debug {
		log.Printf("[DEBUG] - selectHistoryAccount: '%s')", time.Now().Sub(now))
	}

	return result, nil
}

func nilInt64(v sql.NullInt64) *int64 {
	if v.Valid {
		return &v.Int64
	}

	return nil
}
