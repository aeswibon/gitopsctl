package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	tailEventsFile      string
	tailEventsFollow    bool
	tailEventsFromStart bool
	tailEventsPoll      time.Duration
)

var tailEventsCmd = &cobra.Command{
	Use:   "tail-events",
	Short: "Print integration events from a JSONL file (like tail -f)",
	Long: `Reads newline-delimited JSON produced by 'gitopsctl start --events-file'.

By default only prints lines appended after the command starts (--follow).
Use --from-start to emit historical records first.`,
	Example: `
  gitopsctl tail-events --file configs/events.jsonl
  gitopsctl tail-events --file configs/events.jsonl --from-start`,
	Args: cobra.NoArgs,
	RunE: runTailEvents,
}

func runTailEvents(_ *cobra.Command, _ []string) error {
	f, err := os.Open(tailEventsFile)
	if err != nil {
		return fmt.Errorf("open events file: %w", err)
	}
	defer func() { _ = f.Close() }()

	pos := int64(0)
	if !tailEventsFromStart {
		st, err := f.Stat()
		if err != nil {
			return err
		}
		pos = st.Size()
	}

	writeTail := func() error {
		st, err := os.Stat(tailEventsFile)
		if err != nil {
			return err
		}
		if st.Size() < pos {
			_ = f.Close()
			f, err = os.Open(tailEventsFile)
			if err != nil {
				return err
			}
			pos = 0
		}
		if _, err := f.Seek(pos, io.SeekStart); err != nil {
			return err
		}
		buf, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		if len(buf) > 0 {
			if _, err := os.Stdout.Write(buf); err != nil {
				return err
			}
			pos += int64(len(buf))
		}
		return nil
	}

	if err := writeTail(); err != nil {
		return err
	}

	if !tailEventsFollow {
		return nil
	}

	for {
		time.Sleep(tailEventsPoll)
		if err := writeTail(); err != nil {
			return err
		}
	}
}

func init() {
	tailEventsCmd.Flags().StringVar(&tailEventsFile, "file", "configs/events.jsonl", "JSONL file written by --events-file")
	tailEventsCmd.Flags().BoolVar(&tailEventsFollow, "follow", true, "Keep reading as new lines are appended")
	tailEventsCmd.Flags().BoolVar(&tailEventsFromStart, "from-start", false, "Print existing lines from the beginning before following")
	tailEventsCmd.Flags().DurationVar(&tailEventsPoll, "poll-interval", 400*time.Millisecond, "How often to poll the file when following")
	rootCmd.AddCommand(tailEventsCmd)
}
