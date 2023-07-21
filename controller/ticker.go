package controller

import (
	m "arch/model"
	"arch/stream"
	"time"
)

func ticker(events stream.Stream[m.Event]) {
	for tick := range time.NewTicker(time.Second).C {
		events.Push(m.Tick(tick))
	}
}

func (c *controller) handleTick(tick m.Tick) {
	now := time.Time(tick)
	dur := now.Sub(c.prevTick).Seconds()
	c.view.FPS = int(float64(c.frames-1) / dur)
	c.prevTick = now
	c.frames = 0
}
