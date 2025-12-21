// Package bookman: Qutebrowser & multi-browser bookmark/quickmark launcher for ql
package bookman

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/utils"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mitchellh/mapstructure"
	"os"
	"os/exec"
	"strings"
)

// Entry represents a menu entry/bookmark.
type Entry struct {
	Source  string
	Display string
	URL     string
}

const sepString = "********************"

// Register the bookman command at initialization
func init() {
	commands.Register(commands.Command{
		Name:        "bookman",
		Description: "Browser bookmarks & quickmarks manager",
		Run:         Run,
	})
}

// Run implements the bookman logic according to config & sources.
// It aggregates bookmarks/quickmarks/entries from all configured sources
// and opens a selected URL in the browser defined in the global config.
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
			// Show a notification for a failed source, but continue with remaining sources.
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

	// Remove trailing separator(s)
	for len(allEntries) > 0 && allEntries[len(allEntries)-1].Display == sepString {
		allEntries = allEntries[:len(allEntries)-1]
	}
	if len(allEntries) == 0 {
		utils.ShowErrorNotificationWithConfig(&notifCfg, "Bookman", "No bookmarks or quickmarks found!")
		return commands.CommandResult{Success: false}
	}

	// Build menu items for selection (adding group separators, source info, Back if not direct launch)
	var items []string
	if !ctx.IsDirectLaunch() {
		items = append(items, "← Back")
	}
	for _, e := range allEntries {
		if e.Display == sepString {
			items = append(items, sepString)
			continue
		}
		items = append(items, fmt.Sprintf("[%s] %s", e.Source, e.Display))
	}

	// Let the user select an item
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

	// Extract the URL (always the last http(s) word)
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

	// Use the globally configured browser
	browser := ctx.Config().GetBrowser()
	if browser == "" {
		browser = "qutebrowser"
	}
	exec.Command(browser, url).Start()

	return commands.CommandResult{Success: true}
}

// parseSource determines which format parser to call based on source.Format.
func parseSource(src Source) ([]Entry, error) {
	path := utils.ExpandHomeDir(src.Path)
	switch src.Format {
	case "qutebrowser_quickmarks":
		return parseQuteQuickmarks(src.Name, path)
	case "qutebrowser_bookmarks":
		return parseQuteBookmarks(src.Name, path)
	case "chrome_bookmarks_json":
		return parseChromeBookmarksJSON(src.Name, path)
	case "firefox_sqlite":
		return parseFirefoxBookmarks(src.Name, path)
	default:
		return nil, fmt.Errorf("unknown source format: %s", src.Format)
	}
}

// parseQuteQuickmarks parses qutebrowser quickmarks (plain text: <key> <url> per line)
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

// parseQuteBookmarks parses qutebrowser bookmarks (plain text: <url> <title...>)
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

// parseChromeBookmarksJSON parses Chrome/Brave/Chromium bookmarks as found in their Bookmarks JSON file
func parseChromeBookmarksJSON(srcName, path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Recursive structure for folders/bookmarks
	type ChromeItem struct {
		Name     string       `json:"name"`
		URL      string       `json:"url"`
		Type     string       `json:"type"`
		Children []ChromeItem `json:"children"`
	}
	var bm struct {
		Roots struct {
			BookmarkBar ChromeItem `json:"bookmark_bar"`
			Other       ChromeItem `json:"other"`
		} `json:"roots"`
	}

	if err := json.NewDecoder(f).Decode(&bm); err != nil {
		return nil, err
	}

	var result []Entry
	var parse func(folder ChromeItem)
	parse = func(folder ChromeItem) {
		for _, c := range folder.Children {
			if c.Type == "url" && c.URL != "" {
				result = append(result, Entry{
					Source:  srcName,
					Display: fmt.Sprintf("[C] %s - %s", c.Name, c.URL),
					URL:     c.URL,
				})
			} else if c.Type == "folder" && len(c.Children) > 0 {
				parse(c)
			}
		}
	}
	parse(bm.Roots.BookmarkBar)
	parse(bm.Roots.Other)

	return result, nil
}

// parseFirefoxBookmarks parses bookmarks from a Firefox places.sqlite file using go-sqlite3.
// Returns the newest 200 bookmarks with titles.
func parseFirefoxBookmarks(srcName, path string) ([]Entry, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "BOOKMAN DEBUG: sqlite open error for %q: %v\n", path, err)
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	defer db.Close()

	const q = `
	SELECT b.title, p.url
	FROM moz_bookmarks b
	JOIN moz_places p ON b.fk = p.id
	WHERE b.type = 1 AND p.url LIKE 'http%'
	ORDER BY b.dateAdded DESC
	LIMIT 200
	`
	rows, err := db.Query(q)
	if err != nil {
		fmt.Fprintf(os.Stderr, "BOOKMAN DEBUG: sqlite query error: %v\n", err)
		return nil, fmt.Errorf("sqlite query: %w", err)
	}
	defer rows.Close()

	var result []Entry
	count := 0
	for rows.Next() {
		var title, url string
		if err := rows.Scan(&title, &url); err != nil {
			continue
		}
		if title == "" {
			title = "[untitled]"
		}
		count++
		result = append(result, Entry{
			Source:  srcName,
			Display: fmt.Sprintf("[F] %s - %s", title, url),
			URL:     url,
		})
	}
	fmt.Fprintf(os.Stderr, "BOOKMAN DEBUG: Firefox loaded %d entries\n", count)
	if err := rows.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "BOOKMAN DEBUG: rows error: %v\n", err)
		return result, err
	}
	return result, nil
}

// readLines reads a text file into a slice of strings (one per line).
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
