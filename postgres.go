package kronos

import (
	"database/sql"
	"fmt"
	"io"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

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
