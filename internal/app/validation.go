package app

import (
	"fmt"
	"time"

	"github.com/m8urnett/PingGrid/internal/scanner"
	"github.com/m8urnett/toolkit/errors"
)

func validateScanFlags(flags *appFlags) error {
	checks := []struct {
		invalid  bool
		name     string
		value    string
		recovery string
	}{
		{flags.pings < 1 || flags.pings > scanner.MaxPings, "--pings", fmt.Sprintf("%d", flags.pings), fmt.Sprintf("Specify a value from 1 through %d", scanner.MaxPings)},
		{flags.concurrency < 1 || flags.concurrency > scanner.MaxConcurrency, "--concurrency", fmt.Sprintf("%d", flags.concurrency), fmt.Sprintf("Specify a value from 1 through %d", scanner.MaxConcurrency)},
		{flags.Timeout < time.Millisecond || flags.Timeout > maxOperationTime, "--timeout", flags.Timeout.String(), "Specify a duration from 1ms through 1m"},
		{flags.slowThreshold < time.Millisecond || flags.slowThreshold > maxOperationTime, "--slow-threshold", flags.slowThreshold.String(), "Specify a duration from 1ms through 1m"},
		{flags.refresh < 0 || flags.refresh > maxRefreshInterval || (flags.refresh > 0 && flags.refresh < minRefreshInterval), "--refresh", flags.refresh.String(), "Specify 0 or a duration from 100ms through 24h"},
		{flags.rows < 1 || flags.rows > maxGridDimension, "--rows", fmt.Sprintf("%d", flags.rows), fmt.Sprintf("Specify a value from 1 through %d", maxGridDimension)},
		{flags.cols < 1 || flags.cols > maxGridDimension, "--cols", fmt.Sprintf("%d", flags.cols), fmt.Sprintf("Specify a value from 1 through %d", maxGridDimension)},
		{flags.borderWidth < 0 || flags.borderWidth > maxBorderWidth, "--border-width", fmt.Sprintf("%d", flags.borderWidth), fmt.Sprintf("Specify a value from 0 through %d", maxBorderWidth)},
	}
	for _, check := range checks {
		if check.invalid {
			return errors.New(errors.ExitInput, "INPUT_OUT_OF_RANGE", check.name+" is outside the supported range", check.value, check.recovery, nil)
		}
	}
	return validateCanvasDimensions(flags.width, flags.height)
}

func validateCanvasDimensions(width, height int) error {
	if width < 1 || width > maxCanvasDimension {
		return errors.New(errors.ExitInput, "INPUT_OUT_OF_RANGE", "--width is outside the supported range", fmt.Sprintf("%d", width), fmt.Sprintf("Specify a value from 1 through %d", maxCanvasDimension), nil)
	}
	if height < 1 || height > maxCanvasDimension {
		return errors.New(errors.ExitInput, "INPUT_OUT_OF_RANGE", "--height is outside the supported range", fmt.Sprintf("%d", height), fmt.Sprintf("Specify a value from 1 through %d", maxCanvasDimension), nil)
	}
	if int64(width)*int64(height) > maxCanvasPixels {
		return errors.New(errors.ExitInput, "INPUT_TOO_LARGE", "Canvas dimensions exceed the pixel safety limit", fmt.Sprintf("%dx%d", width, height), fmt.Sprintf("Use dimensions totaling no more than %d pixels", maxCanvasPixels), nil)
	}
	return nil
}
