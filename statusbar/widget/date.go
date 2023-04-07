package widget

import (
	"time"
)

const layout string = "Mon 02-01-06 15:04"

type Date struct {
	*Widget
}

func NewDateWidget() *Widget {
	return newWidget("date", 1000, func(widget *Widget) impl {
		return &Date{
			Widget: widget,
		}
	})
}

func (d *Date) setup() {}

func (d *Date) close() {}

func (d *Date) run() {}

func (d *Date) update(block *block) {
	block.FullText = time.Now().Format(layout)
}

func (c *Date) onClick(x int, y int, btn int) {}
