package controller

import (
	m "arch/model"
	"time"
)

func ticker(events m.EventChan) {
	for tick := range time.NewTicker(time.Second).C {
		events <- m.Tick(tick)
	}
}

func (c *controller) handleTick(tick m.Tick) {
	now := time.Time(tick)
	dur := now.Sub(c.prevTick).Seconds()
	c.screen.FPS = int(float64(c.frames-1) / dur)
	c.prevTick = now
	c.frames = 0
}
