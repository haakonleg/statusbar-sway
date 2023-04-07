package widget

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/haakonleg/statusbar-sway/util"
)

const STATFILE = "/proc/stat"

type Cpu struct {
	*Widget
	cpuData   [][]int
	prevIdle  int
	prevTotal int
	statFile  *os.File
}

func NewCpuWidget() *Widget {
	return newWidget("cpu", 4000, func(widget *Widget) impl {
		return &Cpu{
			Widget:    widget,
			cpuData:   make([][]int, 0),
			prevIdle:  -1,
			prevTotal: -1,
		}
	})
}

func (c *Cpu) setup() {
	if statFile, err := os.Open(STATFILE); err != nil {
		log.Fatalf("failed to open %s: %s", STATFILE, err.Error())
	} else {
		c.statFile = statFile
	}
}

func (c *Cpu) close() {
	c.statFile.Close()
}

func (c *Cpu) run() {}

func (c *Cpu) update(block *block) {
	c.readCpuData()

	usagePercent := 0.0

	all := c.cpuData[0]
	// idle + iowait
	idle := all[3] + all[4]
	// total cpu time
	total := util.Sum(all)

	if c.prevIdle != -1 {
		idleDelta := float64(idle - c.prevIdle)
		totalDelta := float64(total - c.prevTotal)
		usagePercent = 100 * (1 - idleDelta/totalDelta)
	}

	c.prevIdle = idle
	c.prevTotal = total

	block.FullText = fmt.Sprintf("CPU %.2f%%", usagePercent)
}

func (c *Cpu) onClick(x int, y int, btn int) {}

func (c *Cpu) readCpuData() {
	c.statFile.Seek(0, io.SeekStart)
	scanner := bufio.NewScanner(c.statFile)

	row := 0
	for scanner.Scan() && strings.HasPrefix(scanner.Text(), "cpu") {
		line := scanner.Text()
		columns := strings.Fields(line)[1:]
		if len(c.cpuData) < row+1 {
			c.cpuData = append(c.cpuData, make([]int, len(columns)))
		}

		// convert to number
		for idx, column := range columns {
			num, _ := strconv.Atoi(column)
			c.cpuData[row][idx] = num
		}

		row++
	}
}
