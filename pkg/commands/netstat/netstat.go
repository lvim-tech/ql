// Package netstat provides network statistics functionality for ql.
// It shows traffic data, active connections, and network information.
package netstat

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lvim-tech/ql/pkg/commands"
	"github.com/lvim-tech/ql/pkg/config"
	"github.com/lvim-tech/ql/pkg/utils"
	"github.com/mitchellh/mapstructure"
)

func init() {
	commands.Register(commands.Command{
		Name:        "netstat",
		Description: "Network statistics",
		Run:         Run,
	})
}

func Run(ctx commands.LauncherContext) commands.CommandResult {
	cfgInterface := ctx.Config().GetNetstatConfig()

	var cfg Config
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           &cfg,
	})
	if err != nil {
		cfg = DefaultConfig()
	} else {
		if decodeErr := decoder.Decode(cfgInterface); decodeErr != nil {
			cfg = DefaultConfig()
		}
	}

	if !cfg.Enabled {
		return commands.CommandResult{
			Success: false,
			Error:   fmt.Errorf("netstat module is disabled in config"),
		}
	}

	notifCfg := ctx.Config().GetNotificationConfig()

	// Check for direct command
	args := ctx.Args()
	if len(args) > 0 {
		return executeDirectCommand(args, &cfg, &notifCfg)
	}

	for {
		var options []string

		if !ctx.IsDirectLaunch() {
			options = append(options, "← Back")
		}

		options = append(options,
			"Current Traffic",
			"Active Connections",
			"Data Usage",
			"Interface Info",
			"Live Monitor",
		)

		choice, err := ctx.Show(options, "Network Statistics")
		if err != nil {
			return commands.CommandResult{Success: false}
		}

		if choice == "← Back" {
			return commands.CommandResult{
				Success: false,
				Error:   commands.ErrBack,
			}
		}

		var actionErr error
		switch choice {
		case "Current Traffic":
			actionErr = showTrafficMenu(ctx, &cfg, &notifCfg)
		case "Active Connections":
			actionErr = showConnections(&notifCfg)
		case "Data Usage":
			actionErr = showDataUsageMenu(ctx, &cfg, &notifCfg)
		case "Interface Info":
			actionErr = showInterfaceInfo(&notifCfg)
		case "Live Monitor":
			actionErr = showLiveMonitor(&cfg, &notifCfg)
		default:
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Netstat Error", fmt.Sprintf("Unknown choice: %s", choice))
			continue
		}

		if actionErr != nil {
			if actionErr.Error() == "cancelled" {
				return commands.CommandResult{Success: false}
			}
			utils.ShowErrorNotificationWithConfig(&notifCfg, "Netstat Error", actionErr.Error())
			continue
		}

		return commands.CommandResult{Success: true}
	}
}

func executeDirectCommand(args []string, cfg *Config, notifCfg *config.NotificationConfig) commands.CommandResult {
	action := strings.ToLower(args[0])

	var err error

	switch action {
	case "traffic":
		period := "today"
		if len(args) > 1 {
			period = args[1]
		}
		err = showTrafficStats(period, "", notifCfg)
	case "connections", "conn":
		err = showConnections(notifCfg)
	case "usage":
		period := "today"
		if len(args) > 1 {
			period = args[1]
		}
		err = showDataUsage(period, "", notifCfg)
	case "info":
		err = showInterfaceInfo(notifCfg)
	case "live":
		err = showLiveMonitor(cfg, notifCfg)
	default:
		err = showTrafficStats(action, "", notifCfg)
	}

	if err != nil {
		return commands.CommandResult{Success: false, Error: err}
	}
	return commands.CommandResult{Success: true}
}

func showTrafficMenu(ctx commands.LauncherContext, _ *Config, notifCfg *config.NotificationConfig) error {
	options := []string{
		"← Back",
		"Today",
		"Yesterday",
		"This Week",
		"This Month",
		"Last Hour",
		"Last 30 Minutes",
	}

	choice, err := ctx.Show(options, "Traffic Period")
	if err != nil {
		return fmt.Errorf("cancelled")
	}

	if choice == "← Back" {
		return fmt.Errorf("cancelled")
	}

	var period string
	switch choice {
	case "Today":
		period = "today"
	case "Yesterday":
		period = "yesterday"
	case "This Week":
		period = "week"
	case "This Month":
		period = "month"
	case "Last Hour":
		period = "1hour"
	case "Last 30 Minutes":
		period = "30min"
	}

	return showTrafficStats(period, "", notifCfg)
}

func showDataUsageMenu(ctx commands.LauncherContext, cfg *Config, notifCfg *config.NotificationConfig) error {
	return showTrafficMenu(ctx, cfg, notifCfg)
}

func showTrafficStats(period string, interfaceName string, _ *config.NotificationConfig) error {
	stats, err := GetNetworkStats(period, interfaceName)
	if err != nil {
		return err
	}

	output := formatTrafficOutput(stats)

	if utils.IsTerminal() {
		fmt.Println(output)
	} else {
		displayStatsGUI(output, "Network Statistics")
	}

	return nil
}

func showDataUsage(period string, interfaceName string, notifCfg *config.NotificationConfig) error {
	return showTrafficStats(period, interfaceName, notifCfg)
}

func showConnections(_ *config.NotificationConfig) error {
	connections, err := getActiveConnections()
	if err != nil {
		return err
	}

	output := formatConnectionsOutput(connections)

	if utils.IsTerminal() {
		fmt.Println(output)
	} else {
		displayStatsGUI(output, "Active Network Connections")
	}

	return nil
}

func showInterfaceInfo(_ *config.NotificationConfig) error {
	interfaces, err := getActiveInterfaces()
	if err != nil {
		return err
	}

	var output strings.Builder
	output.WriteString("Network Interfaces\n\n")

	for _, iface := range interfaces {
		ifaceType := detectInterfaceType(iface)
		status := getInterfaceStatus(iface)
		ip := getInterfaceIP(iface)

		fmt.Fprintf(&output, "┌─ %s (%s - %s)\n", iface, ifaceType, status)

		if ip != "" {
			fmt.Fprintf(&output, "│  IP: %s\n", ip)
		}

		if ifaceType == "wifi" {
			if ssid := getWifiSSID(iface); ssid != "" {
				fmt.Fprintf(&output, "│  SSID: %s\n", ssid)
			}
		}

		output.WriteString("\n")
	}

	if utils.IsTerminal() {
		fmt.Print(output.String())
	} else {
		displayStatsGUI(output.String(), "Network Interfaces")
	}

	return nil
}

func showLiveMonitor(cfg *Config, _ *config.NotificationConfig) error {
	terminal := utils.DetectTerminal()
	if terminal == "" {
		return fmt.Errorf("live monitor requires a terminal")
	}

	script := fmt.Sprintf(`#!/bin/bash
trap 'echo ""; echo "Exiting..."; exit 0' INT TERM
echo "Starting live monitor..."
sleep 1

while true; do
	clear
	echo "╔════════════════════════════════════════════════════════════╗"
	echo "║          Network Live Monitor (Press Ctrl+C to exit)     ║"
	echo "╚════════════════════════════════════════════════════════════╝"
	echo ""
	
	for iface in $(find /sys/class/net/ -maxdepth 1 -type l -printf '%f\n' | grep -v lo); do
		if [ !  -f /sys/class/net/$iface/operstate ]; then
			continue
		fi
		
		state=$(cat /sys/class/net/$iface/operstate 2>/dev/null)
		
		if [[ $iface == wl* ]] || [[ $iface == wlan* ]]; then
			icon="📶"
			type="WiFi"
		elif [[ $iface == eth* ]] || [[ $iface == en* ]]; then
			icon="🔌"
			type="Ethernet"
		else
			icon="🌐"
			type="Other"
		fi
		
		echo "$icon $iface ($type) - $state"
		
		if [ "$state" = "up" ]; then
			ip=$(ip -4 addr show $iface 2>/dev/null | grep -oP '(?<=inet\s)\d+(\.\d+){3}')
			if [ -n "$ip" ]; then
				echo "   IP: $ip"
			fi
			
			if [[ $iface == wl* ]] || [[ $iface == wlan* ]]; then
				if command -v iwgetid &> /dev/null; then
					ssid=$(iwgetid -r $iface 2>/dev/null)
					if [ -n "$ssid" ]; then
						echo "   SSID: $ssid"
					fi
				fi
			fi
			
			rx1=$(cat /sys/class/net/$iface/statistics/rx_bytes 2>/dev/null || echo 0)
			tx1=$(cat /sys/class/net/$iface/statistics/tx_bytes 2>/dev/null || echo 0)
			sleep 1
			rx2=$(cat /sys/class/net/$iface/statistics/rx_bytes 2>/dev/null || echo 0)
			tx2=$(cat /sys/class/net/$iface/statistics/tx_bytes 2>/dev/null || echo 0)
			
			rx_speed=$((rx2 - rx1))
			tx_speed=$((tx2 - tx1))
			
			if [ $rx_speed -gt 1048576 ]; then
				rx_formatted="$(awk "BEGIN {printf \"%%.2f\", $rx_speed/1048576}") MB/s"
			elif [ $rx_speed -gt 1024 ]; then
				rx_formatted="$(awk "BEGIN {printf \"%%.2f\", $rx_speed/1024}") KB/s"
			else
				rx_formatted="$rx_speed B/s"
			fi
			
			if [ $tx_speed -gt 1048576 ]; then
				tx_formatted="$(awk "BEGIN {printf \"%%.2f\", $tx_speed/1048576}") MB/s"
			elif [ $tx_speed -gt 1024 ]; then
				tx_formatted="$(awk "BEGIN {printf \"%%.2f\", $tx_speed/1024}") KB/s"
			else
				tx_formatted="$tx_speed B/s"
			fi
			
			echo "   ↓ Download: $rx_formatted"
			echo "   ↑ Upload:    $tx_formatted"
			
			rx_total=$(cat /sys/class/net/$iface/statistics/rx_bytes 2>/dev/null || echo 0)
			tx_total=$(cat /sys/class/net/$iface/statistics/tx_bytes 2>/dev/null || echo 0)
			
			if [ $rx_total -gt 1073741824 ]; then
				rx_total_formatted="$(awk "BEGIN {printf \"%%.2f\", $rx_total/1073741824}") GB"
			elif [ $rx_total -gt 1048576 ]; then
				rx_total_formatted="$(awk "BEGIN {printf \"%%.2f\", $rx_total/1048576}") MB"
			else
				rx_total_formatted="$(awk "BEGIN {printf \"%%.2f\", $rx_total/1024}") KB"
			fi
			
			if [ $tx_total -gt 1073741824 ]; then
				tx_total_formatted="$(awk "BEGIN {printf \"%%.2f\", $tx_total/1073741824}") GB"
			elif [ $tx_total -gt 1048576 ]; then
				tx_total_formatted="$(awk "BEGIN {printf \"%%.2f\", $tx_total/1048576}") MB"
			else
				tx_total_formatted="$(awk "BEGIN {printf \"%%.2f\", $tx_total/1024}") KB"
			fi
			
			echo "   Total: ↓ $rx_total_formatted  ↑ $tx_total_formatted"
		fi
		
		echo ""
	done
	
	echo "Updated:  $(date '+%%Y-%%m-%%d %%H:%%M:%%S')"
	echo ""
	echo "Press Ctrl+C to exit"
	remaining_sleep=$((%d - 1))
	if [ $remaining_sleep -gt 0 ]; then
		sleep $remaining_sleep
	fi
done
`, cfg.UpdateInterval)

	tmpScript := "/tmp/ql-netstat-live.sh"
	if err := os.WriteFile(tmpScript, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to create monitor script: %w", err)
	}

	cmd := exec.Command(terminal, "-e", "bash", tmpScript)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// displayStatsGUI shows statistics in GUI dialog (yad/zenity/terminal fallback)
func displayStatsGUI(data, title string) error {
	if utils.CommandExists("yad") {
		tmpFile := "/tmp/ql-netstat-data.txt"
		if err := os.WriteFile(tmpFile, []byte(data), 0644); err == nil {
			defer os.Remove(tmpFile)
			cmd := exec.Command("yad",
				"--text-info",
				"--title="+title,
				"--width=800",
				"--height=600",
				"--fontname=Monospace 10",
				"--filename="+tmpFile)
			cmd.Env = os.Environ()
			return cmd.Run()
		}
	}
	if utils.CommandExists("zenity") {
		tmpFile := "/tmp/ql-netstat-data.txt"
		if err := os.WriteFile(tmpFile, []byte(data), 0644); err == nil {
			defer os.Remove(tmpFile)
			cmd := exec.Command("zenity",
				"--text-info",
				"--title="+title,
				"--width=800",
				"--height=600",
				"--filename="+tmpFile)
			cmd.Env = os.Environ()
			return cmd.Run()
		}
	}
	terminal := utils.DetectTerminal()
	if terminal != "" {
		tmpScript := "/tmp/ql-netstat.sh"
		script := fmt.Sprintf("#!/bin/sh\ncat << 'EOF'\n%s\nEOF\necho ''\necho 'Press Enter to close... '\nread\n", data)
		if err := os.WriteFile(tmpScript, []byte(script), 0755); err == nil {
			defer os.Remove(tmpScript)
			return exec.Command(terminal, "-e", tmpScript).Run()
		}
	}
	fmt.Println(data)
	return nil
}

func formatTrafficOutput(stats *NetworkStats) string {
	var output strings.Builder

	fmt.Fprintf(&output, "Network Statistics - %s\n\n", stats.Period)

	for _, iface := range stats.Interfaces {
		icon := getInterfaceIcon(iface.Type)
		statusStr := iface.Status

		if iface.Type == "wifi" && iface.SSID != "" {
			statusStr = fmt.Sprintf("Connected to %s", iface.SSID)
		}

		fmt.Fprintf(&output, "┌─ %s %s (%s - %s)\n", icon, iface.Name, iface.Type, statusStr)

		if iface.IP != "" {
			fmt.Fprintf(&output, "│  IP: %s\n", iface.IP)
		}

		fmt.Fprintf(&output, "│  ↓ Downloaded:     %s\n", FormatBytes(iface.RxBytes))
		fmt.Fprintf(&output, "│  ↑ Uploaded:     %s\n", FormatBytes(iface.TxBytes))
		fmt.Fprintf(&output, "│  Total:          %s\n", FormatBytes(iface.RxBytes+iface.TxBytes))

		duration := stats.EndTime.Sub(stats.StartTime)
		if duration.Seconds() > 0 {
			avgDownSpeed := float64(iface.RxBytes) / duration.Seconds()
			avgUpSpeed := float64(iface.TxBytes) / duration.Seconds()
			fmt.Fprintf(&output, "│  Avg speed:      ↓ %s/s  ↑ %s/s\n",
				FormatBytes(uint64(avgDownSpeed)),
				FormatBytes(uint64(avgUpSpeed)))
		}

		output.WriteString("\n")
	}

	if len(stats.Interfaces) > 1 {
		output.WriteString("Total (all interfaces):\n")
		fmt.Fprintf(&output, "  ↓ Downloaded:  %s\n", FormatBytes(stats.TotalRx))
		fmt.Fprintf(&output, "  ↑ Uploaded:    %s\n", FormatBytes(stats.TotalTx))
		fmt.Fprintf(&output, "  Total:         %s\n", FormatBytes(stats.TotalRx+stats.TotalTx))
	}

	return output.String()
}

func getInterfaceIcon(ifaceType string) string {
	switch ifaceType {
	case "wifi":
		return "📶"
	case "ethernet":
		return "🔌"
	case "vpn":
		return "🔐"
	case "loopback":
		return "🔄"
	default:
		return "🌐"
	}
}

type Connection struct {
	Protocol   string
	LocalAddr  string
	RemoteAddr string
	State      string
	Process    string
}

func getActiveConnections() ([]Connection, error) {
	if !utils.CommandExists("ss") && !utils.CommandExists("netstat") {
		return nil, fmt.Errorf("neither 'ss' nor 'netstat' command found")
	}
	var cmd *exec.Cmd
	if utils.CommandExists("ss") {
		cmd = exec.Command("ss", "-tunap")
	} else {
		cmd = exec.Command("netstat", "-tunap")
	}
	output, err := cmd.Output()
	if err != nil {
		if utils.CommandExists("ss") {
			cmd = exec.Command("ss", "-tuna")
		} else {
			cmd = exec.Command("netstat", "-tuna")
		}
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to get connections: %w", err)
		}
	}
	return parseConnections(string(output)), nil
}

func parseConnections(output string) []Connection {
	var connections []Connection
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		conn := Connection{
			Protocol: fields[0],
		}
		if fields[0] == "tcp" || fields[0] == "udp" {
			if len(fields) >= 5 {
				conn.LocalAddr = fields[4]
				if len(fields) >= 6 {
					conn.RemoteAddr = fields[5]
				}
				if len(fields) >= 2 {
					conn.State = fields[1]
				}
				if len(fields) >= 7 {
					conn.Process = fields[6]
				}
			}
		}
		if conn.Protocol != "" {
			connections = append(connections, conn)
		}
	}
	return connections
}

func formatConnectionsOutput(connections []Connection) string {
	var output strings.Builder

	fmt.Fprintf(&output, "Active Network Connections (%d total)\n\n", len(connections))

	if len(connections) == 0 {
		output.WriteString("No active connections found.\n")
		return output.String()
	}

	tcpConns := 0
	udpConns := 0
	for _, conn := range connections {
		if strings.Contains(strings.ToLower(conn.Protocol), "tcp") {
			tcpConns++
		} else if strings.Contains(strings.ToLower(conn.Protocol), "udp") {
			udpConns++
		}
	}
	fmt.Fprintf(&output, "TCP:  %d connections\n", tcpConns)
	fmt.Fprintf(&output, "UDP: %d connections\n\n", udpConns)

	for _, conn := range connections {
		fmt.Fprintf(&output, "%-6s %-25s → %-25s", conn.Protocol, conn.LocalAddr, conn.RemoteAddr)
		if conn.State != "" {
			fmt.Fprintf(&output, " [%s]", conn.State)
		}
		if conn.Process != "" {
			fmt.Fprintf(&output, " (%s)", conn.Process)
		}
		output.WriteString("\n")
	}
	output.WriteString(fmt.Sprintf("\nGenerated:  %s\n", time.Now().Format("2006-01-02 15:04:05")))
	return output.String()
}
