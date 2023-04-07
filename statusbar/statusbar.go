package statusbar

import (
	"bufio"
	"log"
	"os"
	"time"

	"github.com/haakonleg/statusbar-sway/statusbar/widget"
	"github.com/haakonleg/statusbar-sway/util"
)

type StatusBar struct {
	widgets     []*widget.Widget
	state       []string
	updateQueue chan []*widget.Update
	stdout      *bufio.Writer
}

func NewStatusBar(widgets []*widget.Widget) *StatusBar {
	sb := &StatusBar{
		widgets:     widgets,
		state:       make([]string, len(widgets)),
		updateQueue: make(chan []*widget.Update, 100),
		stdout:      bufio.NewWriter(os.Stdout),
	}

	for idx, widget := range widgets {
		sb.state[idx] = "{}"
		widget.Setup(sb.updateQueue)
	}

	return sb
}

func (s *StatusBar) Run() {
	defer s.stdout.Flush()

	s.stdout.WriteString("{ \"version\": 1, \"click_events\": true }\n")
	s.stdout.WriteString("[\n")

	for _, widget := range s.widgets {
		go widget.Run()
	}

	go s.updateLoop()
	go s.readClickEvent()
	s.mainLoop()

	s.stdout.WriteString("\n]\n")

	for _, widget := range s.widgets {
		widget.Close()
	}
}

// mainLoop receives update signals from the update queue and outputs json to stdout
func (s *StatusBar) mainLoop() {
	for {
		updates, ok := <-s.updateQueue
		if !ok {
			return
		}

		s.stdout.WriteString("[")
		for idx, w := range s.widgets {

			// update json object in state
			for _, update := range updates {
				if update.Widget == w {
					s.state[idx] = update.Json
				}
			}

			// write json object
			s.stdout.WriteString(s.state[idx])

			if idx < len(s.widgets)-1 {
				s.stdout.WriteString(",")
			}
		}
		s.stdout.WriteString("],\n")
		s.stdout.Flush()
	}
}

// updateLoop updates widgets according to their requested interval
func (s *StatusBar) updateLoop() {
	byInterval := make(map[int][]*widget.Widget)
	for _, w := range s.widgets {
		if w.Interval < 1 {
			continue
		}

		if _, exists := byInterval[w.Interval]; exists {
			byInterval[w.Interval] = append(byInterval[w.Interval], w)
		} else {
			byInterval[w.Interval] = []*widget.Widget{w}
		}
	}

	intervals := make([]int, len(byInterval))
	idx := 0
	for interval, _ := range byInterval {
		intervals[idx] = interval
		idx++
	}

	state := make([]int, len(intervals))
	updates := make([]*widget.Update, len(s.widgets))
	prevSleep := 0
	for {
		updateCnt := 0
		sleep := state[0]

		for idx, duration := range state {
			if duration < sleep {
				sleep = duration
			}

			state[idx] -= prevSleep
			if state[idx] == 0 {
				// reset interval
				state[idx] = intervals[idx]

				for _, w := range byInterval[intervals[idx]] {
					updates[updateCnt] = w.Update()
					updateCnt++
				}
			}
		}

		if updateCnt > 0 {
			s.updateQueue <- updates[:updateCnt]
		}

		prevSleep = sleep
		time.Sleep(time.Duration(sleep) * time.Millisecond)
	}
}

// readClickEvent handles click events from stdin
func (s *StatusBar) readClickEvent() {
	buf := make([]byte, 256)

outer:
	for {
		tokenCnt := -1
		startIdx := 0
		endIdx := 0
		event := make([]byte, 0)

		for tokenCnt != 0 {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				log.Printf("failed to read from stdin: %s", err.Error())
				continue outer
			}

			for i := 0; i < n; i++ {
				if buf[i] == '{' {
					if tokenCnt == -1 {
						tokenCnt = 1
					} else {
						tokenCnt += 1
					}

					if startIdx == 0 {
						startIdx = i
					}
				}

				if buf[i] == '}' {
					if tokenCnt > 0 {
						tokenCnt -= 1
					}

					if tokenCnt == 0 {
						endIdx = i
					}
				}
			}

			if tokenCnt == 0 {
				event = append(event, buf[startIdx:endIdx+1]...)
			} else if tokenCnt > 0 {
				if len(event) == 0 {
					event = append(event, buf[startIdx:n]...)
				} else {
					event = append(event, buf[:n]...)
				}
			}
		}

		node, err := util.NewJsonNode(event)
		if err != nil {
			log.Printf("failed to parse event as json: %s", err.Error())
		} else {
			name := node.Get("name").String()
			btn := (int)(node.Get("button").Number())
			x := (int)(node.Get("x").Number())
			y := (int)(node.Get("y").Number())

			// call widget onclick handler
			for _, w := range s.widgets {
				if w.Name == name {
					w.OnClick(x, y, btn)
				}
			}
		}
	}
}
