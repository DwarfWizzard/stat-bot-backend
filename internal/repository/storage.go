package repository

import (
	"time"
)

type Conn struct {
	LastQuery  string
	QuertStart time.Time
	PID        int
}
