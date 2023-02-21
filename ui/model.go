package ui

import "time"

type model struct {
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
