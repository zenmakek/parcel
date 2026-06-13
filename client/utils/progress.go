package utils

import (
	"fmt"
	"io"
	"strings"
	"time"
)

const barWidth = 30

type ProgressReader struct {
	reader    io.Reader
	total     int64
	read      int64
	startTime time.Time
	label     string
}

func NewProgressReader(r io.Reader, total int64, label string) *ProgressReader {
	return &ProgressReader{
		reader:    r,
		total:     total,
		startTime: time.Now(),
		label:     label,
	}
}

func (p *ProgressReader) Read(buf []byte) (int, error) {
	n, err := p.reader.Read(buf)
	p.read += int64(n)
	p.render()
	return n, err
}

func (p *ProgressReader) render() {
	if p.total <= 0 {
		return
	}

	percent := float64(p.read) / float64(p.total)
	if percent > 1.0 {
		percent = 1.0
	}

	filled := int(percent * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	elapsed := time.Since(p.startTime).Seconds()
	speed := float64(p.read)
	if elapsed > 0 {
		speed = float64(p.read) / elapsed
	}

	fmt.Printf("\r  %s [%s] %5.1f%%  %s/s",
		p.label,
		bar,
		percent*100,
		humanSpeed(speed),
	)

	if p.read >= p.total {
		elapsed := time.Since(p.startTime)
		fmt.Printf("\n  Done in %s\n", formatDuration(elapsed))
	}
}

func humanSpeed(bytesPerSec float64) string {
	const (
		KB = 1024
		MB = 1024 * KB
	)

	switch {
	case bytesPerSec >= MB:
		return fmt.Sprintf("%.1f MB", bytesPerSec/MB)
	case bytesPerSec >= KB:
		return fmt.Sprintf("%.1f KB", bytesPerSec/KB)
	default:
		return fmt.Sprintf("%.0f B", bytesPerSec)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}
