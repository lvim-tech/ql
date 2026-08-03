package utils

import (
	"fmt"
	"os"
	"os/exec"
)

// DisplayTextGUI shows a block of text in yad or zenity, falls back to a
// press-enter pager script in a terminal, and prints to stdout as the last
// resort. weather and netstat used to carry near-verbatim copies of this.
//
// Temp files come from os.CreateTemp, never a fixed /tmp name: a predictable
// path another user pre-created as a symlink would be followed and
// overwritten — and the terminal fallback EXECUTES its file.
func DisplayTextGUI(title, data string) error {
	for _, viewer := range []struct {
		bin  string
		args []string
	}{
		{"yad", []string{"--text-info", "--title=" + title, "--width=800", "--height=600", "--fontname=Monospace 10"}},
		{"zenity", []string{"--text-info", "--title=" + title, "--width=800", "--height=600"}},
	} {
		if !CommandExists(viewer.bin) {
			continue
		}
		tmpFile, err := writeTemp("ql-text-*.txt", data, 0644)
		if err != nil {
			break
		}
		defer os.Remove(tmpFile)

		cmd := exec.Command(viewer.bin, append(viewer.args, "--filename="+tmpFile)...)
		cmd.Env = os.Environ()
		return cmd.Run()
	}

	if terminal := DetectTerminal(); terminal != "" {
		script := fmt.Sprintf("#!/bin/sh\ncat << 'EOF'\n%s\nEOF\necho ''\necho 'Press Enter to close... '\nread\n", data)
		if tmpScript, err := writeTemp("ql-text-*.sh", script, 0755); err == nil {
			defer os.Remove(tmpScript)
			return exec.Command(terminal, append(TerminalArgs(terminal), "-e", tmpScript)...).Run()
		}
	}

	fmt.Println(data)
	return nil
}

// writeTemp writes data into a fresh temp file and returns its path.
func writeTemp(pattern, data string, mode os.FileMode) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	if err := os.Chmod(f.Name(), mode); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}
