package util

import (
	"os/exec"
)

func Sum(a []int) int {
	sum := a[0]
	for i := 1; i < len(a); i++ {
		sum += a[i]
	}
	return sum
}

func OpenBrowser(url string) error {
	if err := exec.Command("xdg-open", url).Start(); err != nil {
		return err
	}
	return nil
}
