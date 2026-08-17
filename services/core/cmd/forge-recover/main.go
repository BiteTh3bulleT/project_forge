// forge-recover performs daemon-stopped, whole-store FORGE-K recovery.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"forge/projectforge/services/core/internal/offlinerecovery"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "forge-recover:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("forge-recover", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dataDir := flags.String("data-dir", "", "stopped FORGE daemon data directory")
	bundle := flags.String("bundle", "", "full_backup bundle under the data directory backup root")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	result, err := offlinerecovery.Recover(ctx, offlinerecovery.Request{
		DataDir:    *dataDir,
		BundlePath: *bundle,
	})
	if result != nil {
		if encodeErr := json.NewEncoder(stdout).Encode(result); encodeErr != nil {
			return encodeErr
		}
	}
	return err
}
