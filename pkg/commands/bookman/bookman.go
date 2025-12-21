// Package bookman: Qutebrowser & multi-browser bookmark/quickmark launcher for ql
package bookman

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/utils"
	"github.com/mitchellh/mapstructure"
)

type Entry struct {
	Source  string
	Display string
	URL     string
}

const sepString = "----------"

func init() {
	commands.Register(commands.Command{
		Name:        "bookman",
		Description: "Browser bookmarks & quickmarks manager",
		Run:         Run,
	})
}

// Run implements the bookman logic according to config & sources
func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfgInterface := ctx.Config().GetBookmanConfig()

	var cfg Config
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           &cfg,
	})
	if err != nil {
		cfg = Config{Enabled: true}
	} else {
		if decodeErr := decoder.Decode(cfgInterface); decodeErr != nil {
			cfg = Config{Enabled: true}
		}
	}

	if !cfg.Enabled {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("bookman module is disabled in config"),
		}
	}

	notifCfg := ctx.Config().GetNotificationConfig()

	var allEntries []Entry
	for _, src := range cfg.Sources {
		entries, err := parseSource(src)
		if err != nil {
			// Може да покажеш notification за грешка, но продължаваш!
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Bookman", fmt.Sprintf("Failed: %s (%s)", src.Name, err))
			continue
		}
		if len(entries) > 0 {
			for _, e := range entries {
				allEntries = append(allEntries, Entry{
					Source:  src.Name,
					Display: e.Display,
					URL:     e.URL,
				})
			}
			allEntries = append(allEntries, Entry{Display: sepString})
		}
	}

	// Премахни trailing separator
	for len(allEntries) > 0 && allEntries[len(allEntries)-1].Display == sepString {
		allEntries = allEntries[:len(allEntries)-1]
	}
	if len(allEntries) == 0 {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Bookman", "No bookmarks or quickmarks found!")
		return commands.CommandResult{Success: false}
	}

	var items []string
	if !ctx.IsDirectLaunch() {
		items = append(items, "← Back")
	}
	for _, e := range allEntries {
		// Можеш да маркираш източника
		if e.Display == sepString {
			items = append(items, sepString)
			continue
		}
		items = append(items, fmt.Sprintf("[%s] %s", e.Source, e.Display))
	}

	choice, err := ctx.Show(items, "Bookman")
	if err != nil || choice == "" {
		return commands.CommandResult{Success: false}
	}
	if choice == "← Back" {
		return commands.CommandResult{
			Success: false,
			Error:   commands.ErrBack,
		}
	}
	if choice == sepString {
		return commands.CommandResult{Success: true}
	}

	// Последното "слово" е URL
	url := ""
	fields := strings.Fields(choice)
	for i := len(fields) - 1; i >= 0; i-- {
		f := fields[i]
		if strings.HasPrefix(f, "http://") || strings.HasPrefix(f, "https://") {
			url = f
			break
		}
	}
	if url == "" {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Bookman", "Invalid URL entry!")
		return commands.CommandResult{Success: false}
	}

	browser := ctx.Config().GetBrowser()
	if browser == "" {
		browser = "qutebrowser"
	}
	go func() {
		_ = exec.Command(browser, url).Start()
	}()
	if browser == "" {
		browser = "qutebrowser"
	}
	exec.Command(browser, url).Start()

	return commands.CommandResult{Success: true}
}

// Парсира един source според format
func parseSource(src Source) ([]Entry, error) {
	path := utils.ExpandHomeDir(src.Path)
	switch src.Format {
	case "qutebrowser_quickmarks":
		return parseQuteQuickmarks(src.Name, path)
	case "qutebrowser_bookmarks":
		return parseQuteBookmarks(src.Name, path)
	// case "firefox_sqlite":
	// Тук постави парсинг за Firefox bookmarks със sqlite ако желаеш!
	default:
		return nil, fmt.Errorf("unknown source format: %s", src.Format)
	}
}

func parseQuteQuickmarks(srcName, path string) ([]Entry, error) {
	lines, err := readLines(path)
	if err != nil {
		return nil, err
	}
	var result []Entry
	for _, l := range lines {
		fs := strings.Fields(l)
		if len(fs) >= 2 {
			result = append(result, Entry{
				Source:  srcName,
				Display: fmt.Sprintf("[Q] %s - %s", fs[0], fs[len(fs)-1]),
				URL:     fs[len(fs)-1],
			})
		}
	}
	return result, nil
}

func parseQuteBookmarks(srcName, path string) ([]Entry, error) {
	lines, err := readLines(path)
	if err != nil {
		return nil, err
	}
	var result []Entry
	for _, l := range lines {
		fs := strings.Fields(l)
		if len(fs) >= 2 {
			url := fs[0]
			title := strings.Join(fs[1:], " ")
			result = append(result, Entry{
				Source:  srcName,
				Display: fmt.Sprintf("[B] %s - %s", title, url),
				URL:     url,
			})
		}
	}
	return result, nil
}

// Simple txt file to string slice
func readLines(filename string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}
