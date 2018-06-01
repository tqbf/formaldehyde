// Wrapper functions around go/sql/utils for Postgres
//
// Wants:
//
// POSTGRES_USER (optional)
// POSTGRES_PASSWORD
// POSTGRES_DATABASE
// POSTGRES_HOST

package my

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lib/pq"

	"github.com/jmoiron/sqlx"
)

func MustDbString() string {
	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "postgres"
	}
	host := MustGetenv("POSTGRES_HOST")
	database := MustGetenv("POSTGRES_DATABASE")
	password := MustGetenv("POSTGRES_PASSWORD")

	return fmt.Sprintf("user=%s dbname=%s password=%s host=%s sslmode=disable", user, database, password, host)
}

func MustDbFromEnvironment() *sqlx.DB {
	dbh, err := sqlx.Connect("postgres", MustDbString())
	if err != nil {
		log.Fatal(err)
	}
	return dbh
}

func DbTime(t time.Time) pq.NullTime {
	return pq.NullTime{
		Time:  t,
		Valid: true,
	}
}

func DbBool(v bool) sql.NullBool {
	return sql.NullBool{
		Bool:  v,
		Valid: true,
	}
}

func DbInt(i int) sql.NullInt64 {
	return sql.NullInt64{
		Int64: int64(i),
		Valid: true,
	}
}
