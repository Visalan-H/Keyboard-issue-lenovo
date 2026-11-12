// DeadKeys lets working keys type the characters of broken ones.
//
// Hold CapsLock and press a key to send a different one (layer rules),
// or make a spare key always send another one (direct remaps). The
// rules live in config.json next to the exe.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// version is set at build time with -ldflags "-X main.version=v1.2.3".
var version = "dev"

type options struct {
	debug      bool
	configPath string
}

func main() {
	opts, err := parseFlags(os.Args[1:])
	if err != nil {
		showError(err.Error())
		os.Exit(2)
	}
	os.Exit(run(opts))
}

func parseFlags(args []string) (options, error) {
	var opts options
	var out bytes.Buffer
	fs := flag.NewFlagSet("deadkeys", flag.ContinueOnError)
	fs.SetOutput(&out)
	fs.BoolVar(&opts.debug, "debug", false, "open a console window and log every key event")
	fs.StringVar(&opts.configPath, "config", "", "path to config.json (default: next to deadkeys.exe)")
	showVersion := fs.Bool("version", false, "print the version and exit")
	if err := fs.Parse(args); err != nil {
		return opts, fmt.Errorf("%v\n\n%s", err, out.String())
	}
	if *showVersion {
		showInfo("DeadKeys " + version)
		os.Exit(0)
	}
	if fs.NArg() > 0 {
		return opts, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	if opts.configPath == "" {
		exe, err := os.Executable()
		if err != nil {
			return opts, fmt.Errorf("can't find where deadkeys.exe is: %w", err)
		}
		opts.configPath = filepath.Join(filepath.Dir(exe), "config.json")
	}
	return opts, nil
}
