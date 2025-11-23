package internal

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

// ProgressReporter defines the interface for progress reporting
type ProgressReporter interface {
	SetStatus(status string)
	SetProgress(current, total int)
}

// ConsoleProgress provides a progress bar and status display using progressbar/v3
type ConsoleProgress struct {
	writer    io.Writer
	bar       *progressbar.ProgressBar
	startTime time.Time
	total     int
	status    string
}

// NewConsoleProgress creates a new console progress reporter
func NewConsoleProgress(writer io.Writer) *ConsoleProgress {
	return &ConsoleProgress{
		writer:    writer,
		startTime: time.Now(),
	}
}

// Start begins the progress display
func (p *ConsoleProgress) Start() {
	// Don't create a bar yet - wait for SetStatus or SetProgress
}

// SetStatus updates the current status message
func (p *ConsoleProgress) SetStatus(status string) {
	p.status = status

	// If no progress bar yet, just print the status
	if p.bar == nil {
		fmt.Fprintf(p.writer, "\r\033[K%s", status)
		return
	}

	p.bar.Describe(status)
}

// SetProgress updates the current progress
func (p *ConsoleProgress) SetProgress(current, total int) {
	if total <= 0 {
		return
	}

	// If total changed, create a new progress bar
	if total != p.total {
		p.total = total
		p.bar = nil // Reset bar

		desc := p.status
		if desc == "" {
			desc = "Processing"
		}

		p.bar = progressbar.NewOptions(total,
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionSetWidth(30),
			progressbar.OptionShowCount(),
			progressbar.OptionSetDescription(desc),
			progressbar.OptionSetRenderBlankState(true),
			progressbar.OptionThrottle(50*time.Millisecond),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "█",
				SaucerHead:    ">",
				SaucerPadding: "░",
				BarStart:      "[",
				BarEnd:        "]",
			}),
		)
	}

	if p.bar != nil {
		_ = p.bar.Set(current)
	}
}

// Done marks the progress as complete with a final message
func (p *ConsoleProgress) Done(message string) {
	if p.bar != nil {
		_ = p.bar.Finish()
	}
	elapsed := time.Since(p.startTime).Round(time.Millisecond)
	fmt.Fprintf(p.writer, "\r\033[K\033[32m✓\033[0m %s (took %s)\n", message, elapsed)
}

// Error marks the progress as failed with an error message
func (p *ConsoleProgress) Error(message string) {
	if p.bar != nil {
		_ = p.bar.Finish()
	}
	fmt.Fprintf(p.writer, "\r\033[K\033[31m✗\033[0m %s\n", message)
}

// Stop stops the progress display
func (p *ConsoleProgress) Stop() {
	if p.bar != nil {
		_ = p.bar.Finish()
	}
}
