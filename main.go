package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const version = "0.1.0"
const schemaVersion = 1

type Snapshot struct {
	SchemaVersion int             `json:"schema_version"`
	CapturedAt    string          `json:"captured_at"`
	System        SystemInfo      `json:"system"`
	Git           *GitInfo        `json:"git,omitempty"`
	Tools         map[string]Tool `json:"tools"`
	Env           EnvironmentInfo `json:"env"`
}

type SystemInfo struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Kernel   string `json:"kernel,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

type GitInfo struct {
	Root     string `json:"root,omitempty"`
	Branch   string `json:"branch,omitempty"`
	Commit   string `json:"commit,omitempty"`
	Dirty    bool   `json:"dirty"`
	DiffHash string `json:"diff_hash,omitempty"`
}

type Tool struct {
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
}

type EnvironmentInfo struct {
	Present []string          `json:"present"`
	Safe    map[string]string `json:"safe,omitempty"`
	Path    []string          `json:"path,omitempty"`
}

type Difference struct {
	Severity string `json:"severity"`
	Key      string `json:"key"`
	A        string `json:"a"`
	B        string `json:"b"`
	Why      string `json:"why"`
}

var toolVersionArgs = map[string][]string{
	"git":       {"--version"},
	"go":        {"version"},
	"python":    {"--version"},
	"python3":   {"--version"},
	"node":      {"--version"},
	"npm":       {"--version"},
	"pnpm":      {"--version"},
	"yarn":      {"--version"},
	"bun":       {"--version"},
	"deno":      {"--version"},
	"java":      {"-version"},
	"rustc":     {"--version"},
	"cargo":     {"--version"},
	"docker":    {"--version"},
	"kubectl":   {"version", "--client"},
	"terraform": {"version"},
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "same:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}
	switch args[0] {
	case "capture":
		return captureCommand(args[1:])
	case "compare", "diff":
		return compareCommand(args[1:])
	case "version", "--version", "-v":
		fmt.Println(version)
		return nil
	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\nRun 'same help' for usage", args[0])
	}
}

func printHelp() {
	fmt.Print(`same finds the differences between a working machine and a broken one.

Usage:
  same capture [-o snapshot.json]
  same compare [--json] working.json broken.json
  same version

Examples:
  same capture -o works.json
  same capture -o broken.json
  same compare works.json broken.json

Snapshots never store environment variable values except a small safe allowlist.
`)
}

func captureCommand(args []string) error {
	fs := flag.NewFlagSet("capture", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	output := fs.String("o", "", "write snapshot to a file instead of stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("capture does not accept positional arguments")
	}

	snap := captureSnapshot()
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if *output == "" {
		_, err = os.Stdout.Write(data)
		return err
	}
	if err := os.WriteFile(*output, data, 0o600); err != nil {
		return err
	}
	fmt.Printf("Captured %s\n", *output)
	return nil
}

func compareCommand(args []string) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "print machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return errors.New("usage: same compare [--json] working.json broken.json")
	}
	a, err := readSnapshot(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("read %s: %w", fs.Arg(0), err)
	}
	b, err := readSnapshot(fs.Arg(1))
	if err != nil {
		return fmt.Errorf("read %s: %w", fs.Arg(1), err)
	}
	diffs := compareSnapshots(a, b)
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(diffs)
	}
	printDifferences(diffs)
	return nil
}

func captureSnapshot() Snapshot {
	return Snapshot{
		SchemaVersion: schemaVersion,
		CapturedAt:    time.Now().UTC().Format(time.RFC3339),
		System: SystemInfo{
			OS:       runtime.GOOS,
			Arch:     runtime.GOARCH,
			Kernel:   kernelVersion(),
			Timezone: time.Now().Location().String(),
		},
		Git:   captureGit(),
		Tools: captureTools(),
		Env:   captureEnvironment(),
	}
}

func kernelVersion() string {
	commands := [][]string{}
	switch runtime.GOOS {
	case "windows":
		commands = append(commands, []string{"cmd", "/c", "ver"})
	default:
		commands = append(commands, []string{"uname", "-sr"})
	}
	for _, c := range commands {
		if out, err := exec.Command(c[0], c[1:]...).CombinedOutput(); err == nil {
			return oneLine(string(out))
		}
	}
	return ""
}

func captureGit() *GitInfo {
	root, err := commandOutput("git", "rev-parse", "--show-toplevel")
	if err != nil {
		return nil
	}
	branch, _ := commandOutput("git", "branch", "--show-current")
	commit, _ := commandOutput("git", "rev-parse", "HEAD")
	status, _ := commandOutput("git", "status", "--porcelain=v1", "--untracked-files=normal")
	diff, _ := exec.Command("git", "diff", "--binary", "HEAD").CombinedOutput()
	info := &GitInfo{
		Root:   sanitizePath(root),
		Branch: branch,
		Commit: commit,
		Dirty:  strings.TrimSpace(status) != "",
	}
	if info.Dirty {
		h := sha256.Sum256(diff)
		info.DiffHash = hex.EncodeToString(h[:])
	}
	return info
}

func captureTools() map[string]Tool {
	names := make([]string, 0, len(toolVersionArgs))
	for name := range toolVersionArgs {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make(map[string]Tool)
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		cmd := exec.Command(path, toolVersionArgs[name]...)
		data, _ := cmd.CombinedOutput()
		out[name] = Tool{Path: sanitizePath(path), Version: firstNonEmptyLine(string(data))}
	}
	return out
}

func captureEnvironment() EnvironmentInfo {
	present := make([]string, 0)
	values := make(map[string]string)
	for _, pair := range os.Environ() {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || key == "" {
			continue
		}
		present = append(present, key)
		switch key {
		case "LANG", "LC_ALL", "TZ", "SHELL", "TERM", "COMSPEC", "PROCESSOR_ARCHITECTURE":
			values[key] = sanitizePath(value)
		}
	}
	sort.Strings(present)

	pathParts := []string{}
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p != "" {
			pathParts = append(pathParts, sanitizePath(p))
		}
	}
	return EnvironmentInfo{Present: present, Safe: values, Path: pathParts}
}

func readSnapshot(path string) (Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return Snapshot{}, err
	}
	if s.SchemaVersion != schemaVersion {
		return Snapshot{}, fmt.Errorf("unsupported schema version %d", s.SchemaVersion)
	}
	return s, nil
}

func compareSnapshots(a, b Snapshot) []Difference {
	var diffs []Difference
	add := func(severity, key, av, bv, why string) {
		if av != bv {
			diffs = append(diffs, Difference{Severity: severity, Key: key, A: printable(av), B: printable(bv), Why: why})
		}
	}

	add("high", "system.os", a.System.OS, b.System.OS, "Different operating systems can change paths, binaries, permissions, and system behavior.")
	add("high", "system.arch", a.System.Arch, b.System.Arch, "Different CPU architectures can select different binaries and dependencies.")
	add("medium", "system.kernel", a.System.Kernel, b.System.Kernel, "Kernel differences can affect containers, filesystems, networking, and system calls.")
	add("medium", "system.timezone", a.System.Timezone, b.System.Timezone, "Timezone differences often explain date and scheduling bugs.")

	if a.Git != nil || b.Git != nil {
		var ag, bg GitInfo
		if a.Git != nil {
			ag = *a.Git
		}
		if b.Git != nil {
			bg = *b.Git
		}
		add("high", "git.commit", ag.Commit, bg.Commit, "The two machines are not running the same source revision.")
		add("medium", "git.branch", ag.Branch, bg.Branch, "Different branches may contain different code or configuration.")
		add("high", "git.dirty", fmt.Sprint(ag.Dirty), fmt.Sprint(bg.Dirty), "Uncommitted changes exist on only one side.")
		if ag.Dirty && bg.Dirty {
			add("high", "git.diff", ag.DiffHash, bg.DiffHash, "Both trees are dirty, but their uncommitted changes differ.")
		}
	}

	names := map[string]bool{}
	for n := range a.Tools {
		names[n] = true
	}
	for n := range b.Tools {
		names[n] = true
	}
	sortedNames := make([]string, 0, len(names))
	for n := range names {
		sortedNames = append(sortedNames, n)
	}
	sort.Strings(sortedNames)
	for _, n := range sortedNames {
		at, aok := a.Tools[n]
		bt, bok := b.Tools[n]
		if aok != bok {
			add("high", "tool."+n+".present", fmt.Sprint(aok), fmt.Sprint(bok), "A required executable may be missing on one machine.")
			continue
		}
		if aok {
			add("high", "tool."+n+".version", at.Version, bt.Version, "Different tool versions can change dependency resolution or runtime behavior.")
			add("medium", "tool."+n+".path", at.Path, bt.Path, "The same command resolves to a different executable location.")
		}
	}

	ap := setOf(a.Env.Present)
	bp := setOf(b.Env.Present)
	envNames := map[string]bool{}
	for k := range ap {
		envNames[k] = true
	}
	for k := range bp {
		envNames[k] = true
	}
	envSorted := make([]string, 0, len(envNames))
	for k := range envNames {
		envSorted = append(envSorted, k)
	}
	sort.Strings(envSorted)
	for _, k := range envSorted {
		if isNoisyEnvironmentKey(k) {
			continue
		}
		if ap[k] != bp[k] {
			add("high", "env."+k+".present", fmt.Sprint(ap[k]), fmt.Sprint(bp[k]), "An environment variable exists on only one machine. Values are intentionally not stored.")
		}
	}
	safeNames := map[string]bool{}
	for k := range a.Env.Safe {
		safeNames[k] = true
	}
	for k := range b.Env.Safe {
		safeNames[k] = true
	}
	for k := range safeNames {
		add("medium", "env."+k, a.Env.Safe[k], b.Env.Safe[k], "A non-secret environment setting differs.")
	}
	if strings.Join(a.Env.Path, "\n") != strings.Join(b.Env.Path, "\n") {
		diffs = append(diffs, Difference{
			Severity: "low", Key: "env.PATH", A: strings.Join(a.Env.Path, string(os.PathListSeparator)), B: strings.Join(b.Env.Path, string(os.PathListSeparator)),
			Why: "PATH order can cause the same command name to resolve to a different executable.",
		})
	}

	rank := map[string]int{"high": 0, "medium": 1, "low": 2}
	sort.SliceStable(diffs, func(i, j int) bool {
		if rank[diffs[i].Severity] != rank[diffs[j].Severity] {
			return rank[diffs[i].Severity] < rank[diffs[j].Severity]
		}
		return diffs[i].Key < diffs[j].Key
	})
	return diffs
}

func printDifferences(diffs []Difference) {
	if len(diffs) == 0 {
		fmt.Println("✓ No meaningful differences found.")
		return
	}
	high, medium, low := 0, 0, 0
	for _, d := range diffs {
		switch d.Severity {
		case "high":
			high++
		case "medium":
			medium++
		default:
			low++
		}
	}
	fmt.Printf("Found %d differences (%d high, %d medium, %d low)\n\n", len(diffs), high, medium, low)
	for _, d := range diffs {
		icon := "·"
		switch d.Severity {
		case "high":
			icon = "!"
		case "medium":
			icon = "~"
		}
		fmt.Printf("%s %-6s %s\n", icon, strings.ToUpper(d.Severity), d.Key)
		fmt.Printf("  working: %s\n", truncate(d.A, 140))
		fmt.Printf("  broken:  %s\n", truncate(d.B, 140))
		fmt.Printf("  why:     %s\n\n", d.Why)
	}
}

func commandOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func firstNonEmptyLine(s string) string {
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func sanitizePath(s string) string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		if s == home {
			return "~"
		}
		if strings.HasPrefix(s, home+string(os.PathSeparator)) {
			return "~" + strings.TrimPrefix(s, home)
		}
	}
	return s
}

func isNoisyEnvironmentKey(key string) bool {
	if key == "_" || key == "PWD" || key == "OLDPWD" || key == "SHLVL" || key == "PS1" || key == "PS2" || key == "PROMPT" {
		return true
	}
	for _, prefix := range []string{"TERM_SESSION_", "SSH_", "XDG_", "VTE_", "WT_", "ITERM_", "GPG_TTY"} {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func setOf(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, item := range items {
		m[item] = true
	}
	return m
}

func printable(s string) string {
	if s == "" {
		return "(missing)"
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
