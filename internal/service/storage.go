package service

import (
	"time"
)

type AllertType uint8

const (
	AllertUnattainableDB AllertType = iota
	AllertLowMemory
	AllertManyIdleConn
	AllertTooLongIdleConn
	AllertManyRollbacks
	AllertTooLongQuery
	AllertUnknown
)

var AllertDescription []string = []string{"База данных недостижима", "Осталось мало памяти", "Множество соединений в статусе ожидания", "Соединение слишком долго в статусе ожидания", "Множество ошибок", "Внутренняя ошибка", "Запрос слишком долго в работе", "Неизвестная проблема"}

func (e AllertType) Description() string {
	return AllertDescription[e]
}

type Allert struct {
	Type        AllertType `json:"type"`
	Description string     `json:"description"`
	Data        any        `json:"data"`
}

func NewAllert(e AllertType, data any) *Allert {
	return &Allert{
		Type:        e,
		Description: e.Description(),
		Data:        data,
	}
}

type Conn struct {
	LastQuery     string    `json:"last_query"`
	WaitEvent     *string    `json:"wait_event"`
	WaitEventType *string    `json:"wait_event_type"`
	TxnStart      *time.Time `json:"txn_start"`
	QueryStart    time.Time `json:"query_start"`
	State         *string    `json:"state"`
	PID           int       `json:"pid"`
}

type Metrics struct {
	ServerActive bool `json:"is_active"`

	Rollbacks  float64 `json:"rollbacks"`
	Operations int     `json:"operations"`

	ConnsNum          int     `json:"conns_num"`
	IdleConns         float64 `json:"idle_conns"`
	Conns             []Conn  `json:"conns"`
	LongestActiveConn Conn    `json:"longest_active_conn"`

	MeanResponseTime     float64 `json:"mean_response_time"`
	MaxOperationDuration int     `json:"max_operation_duration"`

	DiskUsage           int64   `json:"disk_usage"`
	DiskUsagePercantage float64 `json:"disk_usage_percantage"`

	Allerts []*Allert `json:"allerts"`
}

type Response struct {
	Data  any   `json:"data"`
	Error error `json:"error"`
}
