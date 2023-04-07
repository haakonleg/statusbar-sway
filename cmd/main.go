package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/haakonleg/statusbar-sway/statusbar"
	"github.com/haakonleg/statusbar-sway/statusbar/widget"
)

var WIDGETS = []*widget.Widget{
	widget.NewWindowTitleWidget(),
	widget.NewWeatherWidget(),
	widget.NewNetworkWidget(),
	widget.NewMemoryWidget(),
	widget.NewCpuWidget(),
	widget.NewDateWidget(),
}

func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetFlags(log.Lmicroseconds)

	homeDir, _ := os.UserHomeDir()
	logFile, err := os.OpenFile(homeDir+"/statusbar.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("failed to open logfile: %s", err.Error())
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	sb := statusbar.NewStatusBar(WIDGETS)
	sb.Run()
}
