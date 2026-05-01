// Package plain is the non-TTY fallback path inside the tuichat binary. When
// the binary is spawned with stdin/stdout not attached to a terminal (piped,
// CI), we skip the Bubble Tea renderer entirely and act as a readline-driven
// shim that still speaks the RPC protocol — so SDK adapters don't need to
// reimplement the fallback per language.
package plain

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/photon-hq/tuichat/internal/protocol"
	"github.com/photon-hq/tuichat/internal/version"
)

const plainSpaceID = "terminal"

// Run handles a plain-mode session: RPC initialization, stdin→notification
// pump, and incoming request dispatch (send/replyToMessage write to stdout,
// everything else is a no-op).
func Run(session *protocol.Session) error {
	var done sync.WaitGroup

	// Log notifications from the adapter's console-hijack: surface to stderr so
	// they don't collide with the stdout protocol stream users see in plain mode.
	session.HandleNotifications(func(method string, raw json.RawMessage) {
		if method != "log" {
			return
		}
		var p protocol.LogNotification
		if err := json.Unmarshal(raw, &p); err != nil {
			return
		}
		level := p.Level
		if level == "" {
			level = "log"
		}
		fmt.Fprintf(os.Stderr, "[%s] %s\n", level, p.Text)
	})

	session.HandleRequests(func(method string, raw json.RawMessage) (any, error) {
		switch method {
		case "initialize":
			return protocol.InitializeResult{
				ProtocolVersion: protocol.Version,
				ServerInfo: protocol.ServerInfo{
					Name:    "tuichat",
					Version: version.Version,
				},
			}, nil

		case "send", "replyToMessage":
			var p protocol.SendParams
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, err
			}
			fmt.Println(plainFormat(p.Content))
			return protocol.SendResult{ID: newID(), Timestamp: iso()}, nil

		case "startTyping", "stopTyping", "reactToMessage", "ensureSpace":
			return nil, nil

		case "shutdown":
			session.Close()
			return nil, nil
		}
		return nil, fmt.Errorf("unknown method: %s", method)
	})

	closed := make(chan struct{})
	session.OnClosed(func() {
		close(closed)
	})

	done.Add(1)
	go func() {
		defer done.Done()
		// Session ending is expected at exit; don't spam stderr on normal close.
		_ = session.Run()
	}()

	if os.Getenv("TUICHAT_QUIET") != "1" {
		fmt.Fprintln(os.Stderr, "[tuichat] non-TTY detected — running in plain readline mode. "+
			"Set TUICHAT_QUIET=1 to silence this notice.")
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	inputDone := make(chan struct{})
	go func() {
		defer close(inputDone)
		for scanner.Scan() {
			line := scanner.Text()
			_ = session.Notify("message", protocol.MessageNotification{
				ID:        newID(),
				SpaceID:   plainSpaceID,
				SenderID:  "terminal-user",
				Content:   protocol.Content{Type: "text", Text: line},
				Timestamp: iso(),
			})
		}
	}()

	// When stdin drains, tell the adapter no more user input is coming — the
	// adapter's `messages` iterator then yields done, the agent's `for await`
	// loop exits, and eventually the adapter issues `shutdown`. We do NOT
	// close the session ourselves here: in-flight `send` responses (including
	// any triggered by the agent's final for-await iteration) must still flow.
	select {
	case <-closed:
		// Adapter closed us (usually via shutdown RPC). Normal exit.
	case <-inputDone:
		_ = session.Notify("streamEnd", nil)
		// Block until the adapter closes the session via shutdown. A hard
		// timeout prevents zombie subprocesses if the agent misbehaves.
		select {
		case <-closed:
		case <-time.After(30 * time.Second):
			session.Close()
		}
	}
	done.Wait()
	return nil
}

// plainFormat mirrors the format used by the pre-tuichat readline provider.
func plainFormat(c protocol.Content) string {
	switch c.Type {
	case "text":
		return c.Text
	case "attachment":
		if c.Size != nil {
			return fmt.Sprintf("[attachment: %s (%d bytes)]", c.Name, *c.Size)
		}
		return fmt.Sprintf("[attachment: %s]", c.Name)
	case "voice":
		name := c.Name
		if name == "" {
			name = "audio"
		}
		if c.Size != nil {
			return fmt.Sprintf("[voice: %s (%d bytes)]", name, *c.Size)
		}
		return fmt.Sprintf("[voice: %s]", name)
	case "contact":
		return "[contact]"
	case "custom":
		if raw, err := json.Marshal(c.Raw); err == nil {
			return "[custom] " + string(raw)
		}
		return "[custom]"
	case "richlink":
		label := c.Title
		if label == "" {
			label = c.Url
		}
		if c.Summary != "" {
			return fmt.Sprintf("[link] %s — %s", label, c.Summary)
		}
		return fmt.Sprintf("[link] %s", label)
	}
	return ""
}

// Supress linter on unused base64 import: keep the import available for future
// inlined content base64 handling without triggering unused-import errors.
var _ = base64.StdEncoding

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func iso() string { return time.Now().UTC().Format(time.RFC3339) }
