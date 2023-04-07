package widget

import (
	"log"
	"sync"

	"github.com/goccy/go-json"
)

// Update is a signal sent to the update queue to update a widget
type Update struct {
	Widget *Widget
	Json   string
}

type block struct {
	sync.Mutex
	Name           string `json:"name"`
	FullText       string `json:"full_text"`
	ShortText      string `json:"short_text,omitempty"`
	Color          string `json:"color,omitempty"`
	Background     string `json:"background,omitempty"`
	Border         string `json:"border,omitempty"`
	BorderTop      int    `json:"border_top,omitempty"`
	BorderBottom   int    `json:"border_bottom,omitempty"`
	BorderLeft     int    `json:"border_left,omitempty"`
	BorderRight    int    `json:"border_right,omitempty"`
	MinWidth       string `json:"min_width,omitempty"`
	Align          string `json:"align,omitempty"`
	Urgent         bool   `json:"urgent,omitempty"`
	Separator      bool   `json:"separator,omitempty"`
	SeparatorWidth int    `json:"separator_block_width,omitempty"`
}

func (b *block) json() string {
	if data, err := json.Marshal(b); err != nil {
		log.Fatalf("failed to encode json: %s", err.Error())
		return "{}"
	} else {
		return string(data)
	}
}

type impl interface {
	setup()
	close()
	run()
	update(*block)
	onClick(int, int, int)
}

type Widget struct {
	impl impl

	Name string

	// interval is the requested update interval for this widget.
	// if set to 0, will not be updated atuomatically. must then
	// signal an update explicitly (via the Update method)
	Interval int

	// current state
	block *block

	// queue is the channel used to signal an update for a widget
	queue chan []*Update
}

func newWidget(name string, interval int, impl func(widget *Widget) impl) *Widget {
	w := &Widget{
		Name:     name,
		Interval: interval,
		block:    &block{Name: name},
	}

	w.impl = impl(w)
	return w
}

func (w *Widget) Setup(queue chan []*Update) {
	w.queue = queue
	w.impl.setup()
}

func (w *Widget) Close() {
	w.impl.close()
}

func (w *Widget) Run() {
	w.impl.run()
}

func (w *Widget) Update() *Update {
	w.block.Lock()
	defer w.block.Unlock()

	w.impl.update(w.block)
	return &Update{Widget: w, Json: w.block.json()}
}

func (w *Widget) OnClick(x int, y int, btn int) {
	log.Printf("onClick %s: %d", w.Name, btn)
	w.impl.onClick(x, y, btn)
}

// sendUpdate is a helper function to signal an update for the widget
func (w *Widget) sendUpdate() {
	w.queue <- []*Update{w.Update()}
}
