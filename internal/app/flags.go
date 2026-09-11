package app

import (
	"time"
)

const (
	stdoutSentinel = ":stdout"
)

const (
	maxGridDimension   = 65536
	maxCanvasDimension = 16384
	maxCanvasPixels    = 64 * 1024 * 1024
	maxBorderWidth     = 64
	maxOperationTime   = time.Minute
	maxRefreshInterval = 24 * time.Hour
	minRefreshInterval = 100 * time.Millisecond
)

type appFlags struct {
	Color          string
	Plain          bool
	Quiet          bool
	Verbose        bool
	Timeout        time.Duration
	scheme         string
	target         string
	ifaceTarget    string
	listInterfaces bool
	allInterfaces  bool
	rows           int
	cols           int
	width          int
	height         int
	borderWidth    int
	outputFormat   string
	outputPath     string
	colorOffline   string
	colorOnline    string
	colorHighlight string
	colorSlow      string
	colorBorder    string
	colorFrame     string
	concurrency    int
	slowThreshold  time.Duration
	refresh        time.Duration
	showVersion    bool
	showExamples   bool
	pings          int
	optimizeOS     bool
	dryRun         bool
	showHUD        bool
	enableARP      bool
	noColor        bool

	// Dedicated output format flags
	optASCII   string
	optJSON    string
	optSummary string
	optHTML    string
	optIframe  string
	optPNG     string
	optList    string
}
