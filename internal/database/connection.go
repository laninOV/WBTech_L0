package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// ConnectDB устанавливает соединение с базой данных.
func (db *DB) ConnectDB() *sql.DB {
	db.name = "postgres"
	connStr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=%s", os.Getenv("user"), os.Getenv("password"), os.Getenv("dbname"), os.Getenv("sslmode"))

	var er error

	sqlDb, er := sql.Open(db.name, connStr)
	if er != nil {
		panic(er)
	}

	er = sqlDb.Ping()
	if er != nil {
		panic(er)
	}

	return sqlDb
}
