package ui

import (
	"scanner/lifecycle"
	"time"
)

type model struct {
	*lifecycle.Lifecycle
	scanStats    []*scanStats
	screenHeight int
	screenWidth  int
	outChan      chan<- any
}

type scanStats struct {
	base            string
	path            string
	start           time.Time
	eta             time.Time
	remaining       time.Duration
	fileProgress    float64
	overallProgress float64
}
