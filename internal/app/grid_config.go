package app

import (
	"github.com/m8urnett/PingGrid/internal/grid"
	"github.com/m8urnett/toolkit/errors"
	"github.com/spf13/cobra"
)

func buildGridConfig(cmd *cobra.Command, flags *appFlags) (grid.GridConfig, error) {
	// Load base color scheme
	gridCfg, err := grid.GetScheme(flags.scheme)
	if err != nil {
		return grid.GridConfig{}, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid color scheme", flags.scheme, "Choose from: dark, light, earth, moss, linen", err)
	}

	// Apply explicit flag overrides if specified
	if cmd.Flags().Changed("color-offline") {
		c, err := grid.ParseHexColor(flags.colorOffline)
		if err != nil {
			return grid.GridConfig{}, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-offline", flags.colorOffline, "Provide valid hex color like #404e41 or 404e41", err)
		}
		gridCfg.ColorOffline = c
	}
	if cmd.Flags().Changed("color-online") {
		c, err := grid.ParseHexColor(flags.colorOnline)
		if err != nil {
			return grid.GridConfig{}, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-online", flags.colorOnline, "Provide valid hex color like #4d86a2 or 4d86a2", err)
		}
		gridCfg.ColorOnline = c
	}
	if cmd.Flags().Changed("color-highlight") {
		c, err := grid.ParseHexColor(flags.colorHighlight)
		if err != nil {
			return grid.GridConfig{}, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-highlight", flags.colorHighlight, "Provide valid hex color like #e4f9d4 or e4f9d4", err)
		}
		gridCfg.ColorHighlight = c
	}
	if cmd.Flags().Changed("color-slow") {
		c, err := grid.ParseHexColor(flags.colorSlow)
		if err != nil {
			return grid.GridConfig{}, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-slow", flags.colorSlow, "Provide valid hex color like #ab7550 or ab7550", err)
		}
		gridCfg.ColorSlow = c
	}
	if cmd.Flags().Changed("color-border") {
		c, err := grid.ParseHexColor(flags.colorBorder)
		if err != nil {
			return grid.GridConfig{}, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-border", flags.colorBorder, "Provide valid hex color like #2c2c2c or 2c2c2c", err)
		}
		gridCfg.ColorBorder = c
	}
	if cmd.Flags().Changed("color-frame") {
		c, err := grid.ParseHexColor(flags.colorFrame)
		if err != nil {
			return grid.GridConfig{}, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-frame", flags.colorFrame, "Provide valid hex color like #2c2c2c or 2c2c2c", err)
		}
		gridCfg.ColorFrame = c
	}

	return gridCfg, nil
}
