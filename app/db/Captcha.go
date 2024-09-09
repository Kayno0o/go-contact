package db

import (
	"time"

	"github.com/uptrace/bun"
)

type Captcha struct {
	bun.BaseModel `bun:"table:player,alias:p"`

	ID        uint64    `bun:",pk,autoincrement"`
	IP        string    `bun:",notnull"`
	Token     string    `bun:",notnull"`
	Value     string    `bun:",notnull"`
	Timestamp time.Time `bun:",notnull,default:current_timestamp"`
}
