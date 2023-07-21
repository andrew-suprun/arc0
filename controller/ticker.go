package controller

import (
	m "arch/model"
	"arch/stream"
	"log"
	"time"
)

func ticker(events stream.Stream[m.Event]) {
	for tick := range time.NewTicker(time.Second).C {
		events.Push(m.Tick(tick))
	}
}

func (c *controller) handleTick(tick m.Tick) {
	now := time.Time(tick)
	dur := now.Sub(c.prevTick)
	seconds := dur.Seconds()
	c.view.FPS = int(float64(c.frames-1) / seconds)
	c.frames = 0
	copied := c.totalCopiedSize + c.fileCopiedSize - c.prevCopied
	c.copySpeed = float64(copied) / (seconds * 1024 * 1024)
	c.prevCopied = c.totalCopiedSize + c.fileCopiedSize
	remainig := c.copySize - c.totalCopiedSize - c.fileCopiedSize
	if copied == 0 {
		c.timeRemaining = 0
	} else {
		c.timeRemaining = time.Duration(remainig * uint64(dur) / copied)
	}

	for _, archive := range c.archives {
		hashed := archive.fileHashed + archive.totalHashed - archive.prevHashed
		archive.speed = float64(hashed) / (seconds * 1024 * 1024)
		archive.prevHashed = archive.fileHashed + archive.totalHashed
		log.Printf("tick: hashed: %d, dur: %v, speed: %7.3f", hashed, seconds, archive.speed)
		remainig := archive.totalSize - archive.fileHashed - archive.totalHashed
		if hashed == 0 {
			archive.timeRemaining = 0
		} else {
			archive.timeRemaining = time.Duration(remainig * uint64(dur) / hashed)
		}
	}

	c.prevTick = now
}
