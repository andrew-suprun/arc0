package ui

import (
	"scanner/lifecycle"
	"time"
)

type model struct {
	*lifecycle.Lifecycle
	screenHeight int
	screenWidth  int
	scanStats    []*scanStats
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
