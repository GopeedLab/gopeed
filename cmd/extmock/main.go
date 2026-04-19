package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
)

type kvFlag map[string]string

func (l *kvFlag) String() string {
	if l == nil {
		return ""
	}
	data, _ := json.Marshal(map[string]string(*l))
	return string(data)
}

func (l *kvFlag) Set(value string) error {
	if *l == nil {
		*l = make(map[string]string)
	}
	idx := -1
	for i := 0; i < len(value); i++ {
		if value[i] == '=' {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return fmt.Errorf("invalid label %q, expected key=value", value)
	}
	(*l)[value[:idx]] = value[idx+1:]
	return nil
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "resolve":
		if err := runResolve(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "eval":
		if err := runEval(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `gopeed extmock CLI

Usage:
  extmock resolve [flags] <extension-dir> <url>
  extmock eval [flags] [script-file]

Flags:
  -label key=value   Request label, repeatable
  -setting key=value Extension setting override, repeatable
  -pretty            Pretty-print JSON output

Examples:
  extmock resolve ./ https://www.google.com
  extmock resolve ./ https://example.com/video -label env=test -setting token=abc
  extmock eval ./test.js
  extmock eval -e "new Blob(['hello']).size"
`)
}

func runResolve(args []string) error {
	fs := flag.NewFlagSet("resolve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var labels kvFlag
	var settings kvFlag
	var pretty bool
	fs.Var(&labels, "label", "Request label in key=value form (repeatable)")
	fs.Var(&settings, "setting", "Extension setting override in key=value form (repeatable)")
	fs.BoolVar(&pretty, "pretty", false, "Pretty-print JSON output")

	flagArgs, positionalArgs, err := splitResolveArgs(args)
	if err != nil {
		return err
	}
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected trailing args: %v", fs.Args())
	}
	if len(positionalArgs) != 2 {
		return fmt.Errorf("resolve expects exactly 2 args: <extension-dir> <url>")
	}

	extDir, err := filepath.Abs(positionalArgs[0])
	if err != nil {
		return err
	}
	url := positionalArgs[1]

	d, cleanup, err := newExtMockDownloader()
	if err != nil {
		return err
	}
	defer cleanup()

	ext, err := d.InstallExtensionByFolder(extDir, true)
	if err != nil {
		return err
	}
	if len(settings) > 0 {
		if err := d.UpdateExtensionSettings(ext.Identity, toAnyMap(settings)); err != nil {
			return err
		}
	}
	result, err := d.Resolve(&base.Request{
		URL:    url,
		Labels: map[string]string(labels),
	}, nil)
	if err != nil {
		return err
	}
	return printJSON(result, pretty)
}

func runEval(args []string) error {
	fs := flag.NewFlagSet("eval", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		script string
		pretty bool
	)
	fs.StringVar(&script, "e", "", "Inline JavaScript to execute")
	fs.StringVar(&script, "eval", "", "Inline JavaScript to execute")
	fs.BoolVar(&pretty, "pretty", false, "Pretty-print JSON output")

	flagArgs, positionalArgs, err := splitEvalArgs(args)
	if err != nil {
		return err
	}
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected trailing args: %v", fs.Args())
	}
	if script == "" && len(positionalArgs) == 0 {
		return fmt.Errorf("eval expects inline code via -e/--eval or a script file path")
	}
	if script != "" && len(positionalArgs) > 0 {
		return fmt.Errorf("eval accepts either inline code or a script file path, not both")
	}
	if len(positionalArgs) > 1 {
		return fmt.Errorf("eval expects at most 1 script file path")
	}

	d, cleanup, err := newExtMockDownloader()
	if err != nil {
		return err
	}
	defer cleanup()

	runtime, err := d.NewExtMockRuntime()
	if err != nil {
		return err
	}
	defer runtime.Close()

	var value any
	if script != "" {
		value, err = runtime.Eval(script)
	} else {
		scriptPath, pathErr := filepath.Abs(positionalArgs[0])
		if pathErr != nil {
			return pathErr
		}
		value, err = runtime.EvalFile(scriptPath)
	}
	if err != nil {
		return err
	}
	return printJSON(value, pretty)
}

func splitResolveArgs(args []string) ([]string, []string, error) {
	var flagArgs []string
	var positionalArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-pretty":
			flagArgs = append(flagArgs, arg)
		case arg == "-label":
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag %q requires a value", arg)
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		case arg == "-setting":
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag %q requires a value", arg)
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		case len(arg) > len("-label=") && arg[:len("-label=")] == "-label=":
			flagArgs = append(flagArgs, arg)
		case len(arg) > len("-setting=") && arg[:len("-setting=")] == "-setting=":
			flagArgs = append(flagArgs, arg)
		case len(arg) > 0 && arg[0] == '-':
			return nil, nil, fmt.Errorf("unknown flag %q", arg)
		default:
			positionalArgs = append(positionalArgs, arg)
		}
	}
	return flagArgs, positionalArgs, nil
}

func splitEvalArgs(args []string) ([]string, []string, error) {
	var flagArgs []string
	var positionalArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-pretty":
			flagArgs = append(flagArgs, arg)
		case arg == "-e", arg == "--eval":
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag %q requires a value", arg)
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		case hasFlagValue(arg, "-e"), hasFlagValue(arg, "--eval"):
			flagArgs = append(flagArgs, arg)
		case len(arg) > 0 && arg[0] == '-':
			return nil, nil, fmt.Errorf("unknown flag %q", arg)
		default:
			positionalArgs = append(positionalArgs, arg)
		}
	}
	return flagArgs, positionalArgs, nil
}

func hasFlagValue(arg string, prefix string) bool {
	return len(arg) > len(prefix)+1 && arg[:len(prefix)+1] == prefix+"="
}

func toAnyMap(values kvFlag) map[string]any {
	result := make(map[string]any, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func newExtMockDownloader() (*download.Downloader, func(), error) {
	return download.NewExtMockDownloader()
}

func printJSON(v any, pretty bool) error {
	var (
		data []byte
		err  error
	)
	if pretty {
		data, err = json.MarshalIndent(v, "", "  ")
	} else {
		data, err = json.Marshal(v)
	}
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
