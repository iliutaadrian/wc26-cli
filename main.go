package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iliutaadrian/wc26/internal/api"
	"github.com/iliutaadrian/wc26/internal/ui"
)

func main() {
	var (
		tokenFlag = flag.String("token", "", "football-data.org API token (overrides env/config)")
		ttl       = flag.Duration("cache", 60*time.Second, "how long to reuse cached API responses")
	)
	flag.Parse()

	token := resolveToken(*tokenFlag)
	if token == "" {
		fmt.Fprintln(os.Stderr, "No API token found.")
		fmt.Fprintln(os.Stderr, "Get a free one at https://www.football-data.org/client/register and then either:")
		fmt.Fprintln(os.Stderr, "  • export WC_API_TOKEN=your_token")
		fmt.Fprintln(os.Stderr, "  • write it to ~/.config/wc26/token")
		fmt.Fprintln(os.Stderr, "  • pass --token your_token")
		os.Exit(1)
	}

	client := api.NewClient(token, *ttl)
	p := tea.NewProgram(ui.New(client), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// resolveToken checks (in order): flag, WC_API_TOKEN env, ~/.config/wc26/token.
func resolveToken(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if env := os.Getenv("WC_API_TOKEN"); env != "" {
		return env
	}
	if home, err := os.UserHomeDir(); err == nil {
		path := filepath.Join(home, ".config", "wc26", "token")
		if data, err := os.ReadFile(path); err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return ""
}
