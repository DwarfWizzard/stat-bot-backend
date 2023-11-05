package repository

import (
	"time"
)

type Conn struct {
	LastQuery     string
	WaitEvent     *string
	WaitEventType *string
	TxnStart      *time.Time
	QueryStart    time.Time
	State         *string
	PID           int
}
