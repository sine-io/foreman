package sqlite

import "database/sql"

type BoardQueryRepository struct {
	db *sql.DB
}

func NewBoardQueryRepository(db *sql.DB) *BoardQueryRepository {
	return &BoardQueryRepository{db: db}
}
