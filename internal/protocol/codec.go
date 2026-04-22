package protocol

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Encode writes an LSP-framed JSON-RPC message: `Content-Length: N\r\n\r\n<body>`.
func Encode(w io.Writer, message any) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode body: %w", err)
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := io.WriteString(w, header); err != nil {
		return err
	}
	if _, err := w.Write(body); err != nil {
		return err
	}
	return nil
}

// ReadOne reads one LSP-framed message body from r and returns the raw JSON body.
// Callers then unmarshal into Request/Notification/Response as appropriate.
func ReadOne(r *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			// blank line = end of headers
			break
		}
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "content-length:") {
			value := strings.TrimSpace(trimmed[len("content-length:"):])
			n, err := strconv.Atoi(value)
			if err != nil || n < 0 {
				return nil, fmt.Errorf("invalid Content-Length %q: %w", value, err)
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		return nil, errors.New("missing Content-Length header")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}
