package netstat

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lvim-tech/ql/pkg/utils"
)

// InterfaceStats represents network statistics for an interface
type InterfaceStats struct {
	Name      string
	Type      string // wifi, ethernet, vpn, loopback
	Status    string // connected, disconnected
	SSID      string // for WiFi
	IP        string
	RxBytes   uint64
	TxBytes   uint64
	RxPackets uint64
	TxPackets uint64
	StartTime time.Time
	EndTime   time.Time
}

// NetworkStats represents statistics for all interfaces
type NetworkStats struct {
	Interfaces []InterfaceStats
	TotalRx    uint64
	TotalTx    uint64
	Period     string
	StartTime  time.Time
	EndTime    time.Time
}

// GetNetworkStats retrieves network statistics for the given period
func GetNetworkStats(period string, interfaceName string) (*NetworkStats, error) {
	start, end, err := parsePeriod(period)
	if err != nil {
		return nil, err
	}

	// Try vnstat first if available and has data
	if utils.CommandExists("vnstat") && vnstatHasData() {
		return getVnstatStats(start, end, interfaceName)
	}

	// Fallback to /sys/class/net (only shows since boot)
	return getSysStats(start, end, interfaceName)
}

// parsePeriod converts period string to start/end time
func parsePeriod(arg string) (time.Time, time.Time, error) {
	now := time.Now()

	// Predefined periods
	switch strings.ToLower(arg) {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return start, now, nil

	case "yesterday":
		yesterday := now.AddDate(0, 0, -1)
		start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
		end := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 0, now.Location())
		return start, end, nil

	case "week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		daysBack := weekday - 1
		start := time.Date(now.Year(), now.Month(), now.Day()-daysBack, 0, 0, 0, 0, now.Location())
		return start, now, nil

	case "month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return start, now, nil
	}

	// Time-based periods:   "30min", "2. 5hours", "3days"
	if matched, _ := regexp.MatchString(`^\d+(\.\d+)?(min|minutes?)$`, arg); matched {
		minutes := parseTimeValue(arg, "min")
		start := now.Add(-time.Duration(minutes) * time.Minute)
		return start, now, nil
	}

	if matched, _ := regexp.MatchString(`^\d+(\.\d+)?(hour|hours?)$`, arg); matched {
		hours := parseTimeValue(arg, "hour")
		start := now.Add(-time.Duration(hours*60) * time.Minute)
		return start, now, nil
	}

	if matched, _ := regexp.MatchString(`^\d+(\.\d+)?(day|days?)$`, arg); matched {
		days := parseTimeValue(arg, "day")
		start := now.AddDate(0, 0, -int(days))
		return start, now, nil
	}

	if matched, _ := regexp.MatchString(`^\d+(\.\d+)?(week|weeks?)$`, arg); matched {
		weeks := parseTimeValue(arg, "week")
		start := now.AddDate(0, 0, -int(weeks*7))
		return start, now, nil
	}

	// Absolute ranges:  "2025-05-12_14:00to2025-05-12_16:30"
	if strings.Contains(arg, "to") {
		parts := strings.Split(arg, "to")
		if len(parts) != 2 {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid range format")
		}

		start, err := parseDateTime(strings.TrimSpace(parts[0]))
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start time: %w", err)
		}

		end, err := parseDateTime(strings.TrimSpace(parts[1]))
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end time: %w", err)
		}

		return start, end, nil
	}

	// Single date: "2025-05-12" (whole day)
	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, arg); matched {
		t, err := time.Parse("2006-01-02", arg)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		end := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
		return start, end, nil
	}

	return time.Time{}, time.Time{}, fmt.Errorf("unknown period format:  %s", arg)
}

func parseTimeValue(s, unit string) float64 {
	// Remove unit suffixes
	s = strings.TrimSuffix(s, "s")
	s = strings.TrimSuffix(s, unit)
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

func parseDateTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)

	// "2025-05-12_14:30"
	if strings.Contains(s, "_") {
		return time.Parse("2006-01-02_15:04", s)
	}

	// "2025-05-12"
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()), nil
}

func vnstatHasData() bool {
	if !utils.FileExists("/var/lib/vnstat/vnstat.db") {
		return false
	}

	cmd := exec.Command("vnstat", "--json", "h")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	var data map[string]any
	if err := json.Unmarshal(output, &data); err != nil {
		return false
	}

	// Check if there's actual traffic data
	interfaces, ok := data["interfaces"].([]any)
	if !ok || len(interfaces) == 0 {
		return false
	}

	return true
}

func getVnstatStats(start, end time.Time, interfaceName string) (*NetworkStats, error) {
	args := []string{"--json", "h"}

	if interfaceName != "" {
		args = append(args, "-i", interfaceName)
	}

	cmd := exec.Command("vnstat", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("vnstat query failed: %w", err)
	}

	var vnstatData struct {
		Interfaces []struct {
			Name    string `json:"name"`
			Traffic struct {
				Hour []struct {
					Date struct {
						Year  int `json:"year"`
						Month int `json:"month"`
						Day   int `json:"day"`
					} `json:"date"`
					Time struct {
						Hour int `json:"hour"`
					} `json:"time"`
					Rx uint64 `json:"rx"`
					Tx uint64 `json:"tx"`
				} `json:"hour"`
			} `json:"traffic"`
		} `json:"interfaces"`
	}

	if err := json.Unmarshal(output, &vnstatData); err != nil {
		return nil, fmt.Errorf("failed to parse vnstat data: %w", err)
	}

	stats := &NetworkStats{
		StartTime: start,
		EndTime:   end,
		Period:    formatPeriod(start, end),
	}

	for _, iface := range vnstatData.Interfaces {
		ifaceStats := InterfaceStats{
			Name:      iface.Name,
			Type:      detectInterfaceType(iface.Name),
			Status:    getInterfaceStatus(iface.Name),
			StartTime: start,
			EndTime:   end,
		}

		if ifaceStats.Type == "wifi" {
			ifaceStats.SSID = getWifiSSID(iface.Name)
		}

		ifaceStats.IP = getInterfaceIP(iface.Name)

		// Sum traffic within the time range
		for _, hour := range iface.Traffic.Hour {
			hourTime := time.Date(hour.Date.Year, time.Month(hour.Date.Month), hour.Date.Day,
				hour.Time.Hour, 0, 0, 0, time.Local)

			if (hourTime.After(start) && hourTime.Before(end)) || hourTime.Equal(start) {
				ifaceStats.RxBytes += hour.Rx
				ifaceStats.TxBytes += hour.Tx
			}
		}

		stats.Interfaces = append(stats.Interfaces, ifaceStats)
		stats.TotalRx += ifaceStats.RxBytes
		stats.TotalTx += ifaceStats.TxBytes
	}

	return stats, nil
}

func getSysStats(start, end time.Time, interfaceName string) (*NetworkStats, error) {
	interfaces, err := getActiveInterfaces()
	if err != nil {
		return nil, err
	}

	stats := &NetworkStats{
		StartTime: start,
		EndTime:   end,
		Period:    formatPeriod(start, end) + " (since boot)",
	}

	for _, iface := range interfaces {
		if interfaceName != "" && iface != interfaceName {
			continue
		}

		ifaceStats := InterfaceStats{
			Name:      iface,
			Type:      detectInterfaceType(iface),
			Status:    getInterfaceStatus(iface),
			StartTime: start,
			EndTime:   end,
		}

		if ifaceStats.Type == "wifi" {
			ifaceStats.SSID = getWifiSSID(iface)
		}

		ifaceStats.IP = getInterfaceIP(iface)

		// Read from /sys/class/net
		rxPath := filepath.Join("/sys/class/net", iface, "statistics", "rx_bytes")
		txPath := filepath.Join("/sys/class/net", iface, "statistics", "tx_bytes")

		if rxData, err := os.ReadFile(rxPath); err == nil {
			ifaceStats.RxBytes, _ = strconv.ParseUint(strings.TrimSpace(string(rxData)), 10, 64)
		}

		if txData, err := os.ReadFile(txPath); err == nil {
			ifaceStats.TxBytes, _ = strconv.ParseUint(strings.TrimSpace(string(txData)), 10, 64)
		}

		stats.Interfaces = append(stats.Interfaces, ifaceStats)
		stats.TotalRx += ifaceStats.RxBytes
		stats.TotalTx += ifaceStats.TxBytes
	}

	return stats, nil
}

func getActiveInterfaces() ([]string, error) {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return nil, err
	}

	var interfaces []string
	for _, entry := range entries {
		name := entry.Name()
		if name != "lo" { // Skip loopback
			interfaces = append(interfaces, name)
		}
	}

	return interfaces, nil
}

func detectInterfaceType(name string) string {
	if strings.HasPrefix(name, "wl") || strings.HasPrefix(name, "wlan") {
		return "wifi"
	}
	if strings.HasPrefix(name, "eth") || strings.HasPrefix(name, "en") {
		return "ethernet"
	}
	if strings.HasPrefix(name, "tun") || strings.HasPrefix(name, "vpn") {
		return "vpn"
	}
	if name == "lo" {
		return "loopback"
	}
	return "unknown"
}

func getInterfaceStatus(name string) string {
	operstatePath := filepath.Join("/sys/class/net", name, "operstate")
	data, err := os.ReadFile(operstatePath)
	if err != nil {
		return "unknown"
	}

	state := strings.TrimSpace(string(data))
	if state == "up" {
		return "connected"
	}
	return "disconnected"
}

func getWifiSSID(interfaceName string) string {
	if !utils.CommandExists("iwgetid") {
		return ""
	}

	cmd := exec.Command("iwgetid", "-r", interfaceName)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

func getInterfaceIP(name string) string {
	if !utils.CommandExists("ip") {
		return ""
	}

	cmd := exec.Command("ip", "-4", "addr", "show", name)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse IP from:  "inet 192.168.1.100/24 brd..."
	for line := range strings.SplitSeq(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "inet ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				// Remove subnet mask
				ip := strings.Split(fields[1], "/")[0]
				return ip
			}
		}
	}

	return ""
}

func formatPeriod(start, end time.Time) string {
	duration := end.Sub(start)

	if start.Year() == end.Year() && start.Month() == end.Month() && start.Day() == end.Day() {
		// Same day
		return start.Format("Jan 02, 2006") + " " + start.Format("15:04") + " - " + end.Format("15:04")
	}

	// Different days
	durationStr := formatDuration(duration)
	return start.Format("Jan 02") + " - " + end.Format("Jan 02, 2006") + " (" + durationStr + ")"
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 24 {
		days := hours / 24
		hours = hours % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}

	return fmt.Sprintf("%dm", minutes)
}

// FormatBytes converts bytes to human-readable format
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
