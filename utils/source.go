package utils

import (
	"time"
)

type Source interface {
	Collect(chan<- Record, time.Time) error
}
