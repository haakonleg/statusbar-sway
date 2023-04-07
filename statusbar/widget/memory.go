package widget

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

const MEMFILE = "/proc/meminfo"

type Memory struct {
	*Widget
	memTotal int
	memAvail int
	memFile  *os.File
}

func NewMemoryWidget() *Widget {
	return newWidget("memory", 4000, func(widget *Widget) impl {
		return &Memory{
			Widget: widget,
		}
	})
}

func (m *Memory) setup() {
	if memFile, err := os.Open(MEMFILE); err != nil {
		log.Fatalf("failed to open %s: %s", MEMFILE, err.Error())
	} else {
		m.memFile = memFile
	}
}

func (m *Memory) close() {
	m.memFile.Close()
}

func (m *Memory) run() {}

func (m *Memory) update(block *block) {
	m.readMemoryData()
	memTotalGb := float64(m.memTotal) / (1024 * 1024)
	memUsedGb := float64(m.memTotal-m.memAvail) / (1024 * 1024)

	block.MinWidth = "00.0/00GiB"
	block.FullText = fmt.Sprintf("MEM %.1f/%.0fGiB", memUsedGb, memTotalGb)
}

func (c *Memory) onClick(x int, y int, btn int) {}

func (m *Memory) readMemoryData() {
	m.memFile.Seek(0, io.SeekStart)
	scanner := bufio.NewScanner(m.memFile)

	m.memTotal = -1
	m.memAvail = -1

	for scanner.Scan() && (m.memTotal == -1 || m.memAvail == -1) {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal") {
			value := strings.Fields(line)[1]
			m.memTotal, _ = strconv.Atoi(value)
		} else if strings.HasPrefix(line, "MemAvailable") {
			value := strings.Fields(line)[1]
			m.memAvail, _ = strconv.Atoi(value)
		}
	}
}
