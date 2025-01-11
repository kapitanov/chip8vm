package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/kapitanov/chip8vm/internal/hal"
	"github.com/kapitanov/chip8vm/internal/vm"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:           fmt.Sprintf("%s PATH_TO_ROM_FILE", filepath.Base(os.Args[0])),
		Short:         "Run emulator",
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
	}

	verbose := cmd.Flags().BoolP("verbose", "v", false, "enable verbose logging")

	cmd.RunE = func(_ *cobra.Command, args []string) error {
		loggerOpts := &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}
		if *verbose {
			loggerOpts.Level = slog.LevelDebug
		}

		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, loggerOpts)))

		path := args[0]
		bs, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("unable to load file %q: %w", path, err)
		}

		h, err := hal.New()
		if err != nil {
			return fmt.Errorf("unable to initialize hal: %w", err)
		}
		defer h.Shutdown()

		machine := vm.New(bs)

		for {
			err = machine.Run(h)

			if errors.Is(err, hal.ErrQuit) {
				return nil
			}

			if errors.Is(err, hal.ErrReboot) {
				continue
			}

			return nil
		}
	}

	cmd.SetArgs(os.Args[1:])
	if err := cmd.Execute(); err != nil {
		slog.Error("fatal error", "err", err)
		os.Exit(1)
	}
}
