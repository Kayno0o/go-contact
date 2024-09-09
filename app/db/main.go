package db

import (
	"context"
	"database/sql"
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bundebug"
)

var (
	DB  *bun.DB
	Ctx = context.Background()
)

func Init() {
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}
	DB = bun.NewDB(sqldb, sqlitedialect.New())

	file, err := os.Create("bundebug.log")
	if err != nil {
		panic(err)
	}

	DB.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.WithWriter(file),
	))

	if err := DB.Ping(); err != nil {
		panic(err)
	}

	DB.RegisterModel(&Captcha{})
	if _, err = DB.NewCreateTable().Model(&Captcha{}).Exec(Ctx); err != nil {
		panic(err)
	}
}
