package output

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/term"
)

// DetectPager returns the preferred pager found on PATH.
func DetectPager() string {
	if _, err := exec.LookPath("less"); err == nil {
		return "less"
	}
	if _, err := exec.LookPath("more"); err == nil {
		return "more"
	}
	return ""
}

// RunPager executes the pager and writes content to its stdin.
func RunPager(pager string, content string) error {
	args := []string{}
	if pager == "less" {
		args = []string{"-R"}
	}
	cmd := exec.Command(pager, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return err
	}
	if _, err := stdin.Write([]byte(content)); err != nil {
		_ = stdin.Close()
		// Try to wait to reap process before returning
		_ = cmd.Wait()
		return err
	}
	if err := stdin.Close(); err != nil {
		// Still wait; report composite error
		_ = cmd.Wait()
		return fmt.Errorf("stdin close: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

// ClearTerminal tries to clear the screen and scrollback buffer using cross-platform methods.
func ClearTerminal() {
	// Use ANSI escape sequences if we're on a terminal
	if term.IsTerminal(int(os.Stdout.Fd())) {
		// ANSI sequences: clear screen, clear scrollback (3J), move cursor home
		fmt.Print("\033[2J\033[3J\033[H")
	}

	// Best-effort external clear command (cross-platform)
	clearCmd := GetClearCommand()
	if clearCmd != "" {
		_ = exec.Command(clearCmd).Run()
	}
}

// GetClearCommand returns the appropriate clear command for the current platform.
func GetClearCommand() string {
	// On Windows, try 'cls' first, then 'clear' (for WSL/Git Bash)
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("cls"); err == nil {
			return "cls"
		}
	}

	// On Unix-like systems (and as fallback on Windows), try 'clear'
	if _, err := exec.LookPath("clear"); err == nil {
		return "clear"
	}

	return ""
}
