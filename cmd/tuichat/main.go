package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"golang.org/x/term"

	"github.com/photon-hq/tuichat/internal/plain"
	"github.com/photon-hq/tuichat/internal/protocol"
	"github.com/photon-hq/tuichat/internal/server"
	"github.com/photon-hq/tuichat/internal/store"
	"github.com/photon-hq/tuichat/internal/ui"
	"github.com/photon-hq/tuichat/internal/version"
)

func main() {
	connectFlag := flag.String("connect", "", "HOST:PORT for the adapter's JSON-RPC listener (required)")
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("tuichat %s\n", version.Version)
		return
	}
	if *connectFlag == "" {
		fmt.Fprintln(os.Stderr, "tuichat: --connect HOST:PORT is required. See --help.")
		os.Exit(2)
	}

	if err := run(*connectFlag); err != nil {
		fmt.Fprintf(os.Stderr, "[tuichat] fatal: %v\n", err)
		os.Exit(1)
	}
}

func isInteractive() bool {
	if os.Getenv("TUICHAT_FORCE_PLAIN") == "1" {
		return false
	}
	if os.Getenv("TUICHAT_FORCE_TUI") == "1" {
		return true
	}
	return term.IsTerminal(int(os.Stdin.Fd())) &&
		term.IsTerminal(int(os.Stdout.Fd()))
}

func run(addr string) error {
	if !strings.Contains(addr, ":") {
		return fmt.Errorf("invalid --connect value: %s", addr)
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}

	session := protocol.NewSession(conn)

	if !isInteractive() {
		return plain.Run(session)
	}

	return runRich(session)
}

func runRich(session *protocol.Session) error {
	zone.NewGlobal()

	st := store.New(nil)
	model := ui.NewModel(st)

	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	srv := &server.Server{
		Store:   st,
		Session: session,
		Program: program,
	}

	session.HandleRequests(func(method string, params json.RawMessage) (any, error) {
		return srv.HandleRequest(method, params)
	})
	session.HandleNotifications(func(method string, params json.RawMessage) {
		srv.HandleNotification(method, params)
	})
	session.OnClosed(func() {
		program.Quit()
	})

	go srv.PumpUserInput()
	go func() {
		_ = session.Run()
	}()

	defer func() {
		st.CloseInput()
		session.Close()
	}()

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("program: %w", err)
	}
	return nil
}
