package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Polaris-F/watchpid/internal/buildinfo"
	"github.com/Polaris-F/watchpid/internal/config"
	"github.com/Polaris-F/watchpid/internal/model"
	"github.com/Polaris-F/watchpid/internal/notify"
	"github.com/Polaris-F/watchpid/internal/store"
	"github.com/Polaris-F/watchpid/internal/watch"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "version", "-v", "--version":
		return runVersion(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printHelp(stdout)
		return 0
	}

	s, err := store.New("")
	if err != nil {
		return printError(stderr, err)
	}
	service := watch.NewService(s)

	switch args[0] {
	case "watch":
		return runWatch(service, args[1:], stdout, stderr)
	case "run":
		return runCommand(service, args[1:], stdout, stderr)
	case "status":
		return runStatus(s, args[1:], stdout, stderr)
	case "list":
		return runList(s, args[1:], stdout, stderr)
	case "cancel":
		return runCancel(service, args[1:], stdout, stderr)
	case "notify":
		return runNotify(s, args[1:], stdout, stderr)
	case "daemon":
		return runDaemon(service, args[1:], stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printHelp(stderr)
		return 1
	}
}

func runVersion(args []string, stdout io.Writer, stderr io.Writer) int {
	args, jsonLate := extractBoolFlag(args, "--json")

	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if jsonLate {
		*jsonOut = true
	}

	info := buildinfo.Current()
	if *jsonOut {
		return printJSON(stdout, info)
	}
	fmt.Fprintln(stdout, info.String())
	return 0
}

func runWatch(service *watch.Service, args []string, stdout io.Writer, stderr io.Writer) int {
	args, detachLate := extractBoolFlag(args, "--detach")
	args, jsonLate := extractBoolFlag(args, "--json")

	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	name := fs.String("name", "", "optional display name")
	detach := fs.Bool("detach", false, "run watcher in background")
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: watchpid watch <pid> [--name name] [--detach] [--json]")
		return 2
	}
	if detachLate {
		*detach = true
	}
	if jsonLate {
		*jsonOut = true
	}

	var pid int
	if _, err := fmt.Sscanf(fs.Arg(0), "%d", &pid); err != nil {
		return printError(stderr, fmt.Errorf("invalid pid: %w", err))
	}

	w, err := service.RegisterExisting(pid, *name, *detach)
	if err != nil {
		return printCommandError(*jsonOut, stdout, stderr, err)
	}
	return printWatch(*jsonOut, stdout, w)
}

func runCommand(service *watch.Service, args []string, stdout io.Writer, stderr io.Writer) int {
	commandIndex := -1
	for i, arg := range args {
		if arg == "--" {
			commandIndex = i
			break
		}
	}
	if commandIndex == -1 {
		fmt.Fprintln(stderr, "usage: watchpid run [--name name] [--detach] [--json] -- <command> [args...]")
		return 2
	}

	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	name := fs.String("name", "", "optional display name")
	detach := fs.Bool("detach", false, "run watcher in background")
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args[:commandIndex]); err != nil {
		return 2
	}

	command := args[commandIndex+1:]
	var commandStdout io.Writer
	var commandStderr io.Writer
	if !*detach && !*jsonOut {
		commandStdout = stdout
		commandStderr = stderr
	}

	w, err := service.RegisterRun(*name, command, *detach, commandStdout, commandStderr)
	if err != nil {
		return printCommandError(*jsonOut, stdout, stderr, err)
	}
	return printWatch(*jsonOut, stdout, w)
}

func runStatus(s *store.Store, args []string, stdout io.Writer, stderr io.Writer) int {
	args, jsonLate := extractBoolFlag(args, "--json")

	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: watchpid status <watch_id> [--json]")
		return 2
	}
	if jsonLate {
		*jsonOut = true
	}

	w, err := s.LoadWatch(fs.Arg(0))
	if err != nil {
		return printCommandError(*jsonOut, stdout, stderr, err)
	}
	return printWatch(*jsonOut, stdout, w)
}

func runList(s *store.Store, args []string, stdout io.Writer, stderr io.Writer) int {
	args, jsonLate := extractBoolFlag(args, "--json")

	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if jsonLate {
		*jsonOut = true
	}

	items, err := s.ListWatches()
	if err != nil {
		return printCommandError(*jsonOut, stdout, stderr, err)
	}

	if *jsonOut {
		return printJSON(stdout, items)
	}

	if len(items) == 0 {
		fmt.Fprintln(stdout, "No watches found.")
		return 0
	}

	for _, item := range items {
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\n",
			item.ID,
			item.Status,
			item.Mode,
			item.CreatedAt.Format(time.RFC3339),
			item.Name,
		)
	}
	return 0
}

func runCancel(service *watch.Service, args []string, stdout io.Writer, stderr io.Writer) int {
	args, jsonLate := extractBoolFlag(args, "--json")

	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: watchpid cancel <watch_id> [--json]")
		return 2
	}
	if jsonLate {
		*jsonOut = true
	}

	w, err := service.Cancel(fs.Arg(0))
	if err != nil {
		return printCommandError(*jsonOut, stdout, stderr, err)
	}
	return printWatch(*jsonOut, stdout, w)
}

func runNotify(s *store.Store, args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: watchpid notify <test|setup>")
		return 2
	}

	switch args[0] {
	case "test":
		parsedArgs, jsonLate := extractBoolFlag(args[1:], "--json")

		fs := flag.NewFlagSet("notify test", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "print JSON")
		if err := fs.Parse(parsedArgs); err != nil {
			return 2
		}
		if jsonLate {
			*jsonOut = true
		}
		cfg, err := config.Load(s)
		if err != nil {
			return printCommandError(*jsonOut, stdout, stderr, err)
		}
		if strings.TrimSpace(cfg.Notify.PushPlus.Token) == "" {
			return printCommandError(*jsonOut, stdout, stderr, errors.New(config.MissingTokenHint(s)))
		}
		n := notify.PushPlus{Token: cfg.Notify.PushPlus.Token}
		msg := notify.Message{
			Title: "watchpid test",
			Body:  "watchpid test message sent at " + time.Now().Format(time.RFC3339),
		}
		if err := n.Send(context.Background(), msg); err != nil {
			return printCommandError(*jsonOut, stdout, stderr, err)
		}
		if *jsonOut {
			return printJSON(stdout, map[string]string{"status": "sent"})
		}
		fmt.Fprintln(stdout, "Test notification sent.")
		return 0
	case "setup":
		fs := flag.NewFlagSet("notify setup", flag.ContinueOnError)
		fs.SetOutput(stderr)
		tokenFlag := fs.String("token", "", "PushPlus token")
		jsonOut := fs.Bool("json", false, "print JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}

		token := strings.TrimSpace(*tokenFlag)
		if token == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Fprintf(stdout, "PushPlus token [%s]: ", s.ConfigPath)
			value, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				return printError(stderr, err)
			}
			token = strings.TrimSpace(value)
		}
		if token == "" {
			return printCommandError(*jsonOut, stdout, stderr, fmt.Errorf("empty token"))
		}
		if err := config.SavePushPlusToken(s, token); err != nil {
			return printCommandError(*jsonOut, stdout, stderr, err)
		}
		if *jsonOut {
			return printJSON(stdout, map[string]string{
				"status":      "saved",
				"config_path": s.ConfigPath,
			})
		}
		fmt.Fprintf(stdout, "Saved PushPlus token to %s\n", s.ConfigPath)
		return 0
	default:
		fmt.Fprintln(stderr, "usage: watchpid notify <test|setup>")
		return 2
	}
}

func runDaemon(service *watch.Service, args []string, stderr io.Writer) int {
	if len(args) == 0 {
		return printError(stderr, fmt.Errorf("missing daemon subcommand"))
	}
	fs := flag.NewFlagSet("daemon", flag.ContinueOnError)
	fs.SetOutput(stderr)
	id := fs.String("id", "", "watch id")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if strings.TrimSpace(*id) == "" {
		return printError(stderr, fmt.Errorf("missing --id"))
	}

	switch args[0] {
	case "watch":
		if err := service.MonitorExisting(*id); err != nil {
			return printError(stderr, err)
		}
		return 0
	case "run":
		if err := service.RunRegistered(*id, nil, nil); err != nil {
			return printError(stderr, err)
		}
		return 0
	default:
		return printError(stderr, fmt.Errorf("unknown daemon subcommand: %s", args[0]))
	}
}

func printWatch(jsonOut bool, stdout io.Writer, w *model.Watch) int {
	if jsonOut {
		return printJSON(stdout, w)
	}

	fmt.Fprintf(stdout, "ID: %s\n", w.ID)
	fmt.Fprintf(stdout, "Name: %s\n", w.Name)
	fmt.Fprintf(stdout, "Mode: %s\n", w.Mode)
	fmt.Fprintf(stdout, "Status: %s\n", w.Status)
	if w.TargetPID > 0 {
		fmt.Fprintf(stdout, "PID: %d\n", w.TargetPID)
	}
	if w.WatcherPID > 0 {
		fmt.Fprintf(stdout, "WatcherPID: %d\n", w.WatcherPID)
	}
	if w.LogPath != "" {
		fmt.Fprintf(stdout, "Log: %s\n", w.LogPath)
	}
	if w.NotificationError != "" {
		fmt.Fprintf(stdout, "Notification: %s\n", w.NotificationError)
	}
	return 0
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "watchpid")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  watch <pid> [--name name] [--detach] [--json]")
	fmt.Fprintln(w, "  run [--name name] [--detach] [--json] -- <command> [args...]")
	fmt.Fprintln(w, "  status <watch_id> [--json]")
	fmt.Fprintln(w, "  list [--json]")
	fmt.Fprintln(w, "  cancel <watch_id> [--json]")
	fmt.Fprintln(w, "  notify test [--json]")
	fmt.Fprintln(w, "  notify setup [--token xxx] [--json]")
	fmt.Fprintln(w, "  version [--json]")
}

func printJSON(w io.Writer, payload interface{}) int {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "json error: %v\n", err)
		return 1
	}
	fmt.Fprintln(w, string(data))
	return 0
}

func printCommandError(jsonOut bool, stdout io.Writer, stderr io.Writer, err error) int {
	if jsonOut {
		return printJSON(stdout, map[string]string{"error": err.Error()})
	}
	return printError(stderr, err)
}

func printError(stderr io.Writer, err error) int {
	fmt.Fprintln(stderr, err)
	return 1
}

func extractBoolFlag(args []string, flagName string) ([]string, bool) {
	out := make([]string, 0, len(args))
	enabled := false
	for _, arg := range args {
		if arg == flagName {
			enabled = true
			continue
		}
		out = append(out, arg)
	}
	return out, enabled
}
