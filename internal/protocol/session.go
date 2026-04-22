package protocol

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// RequestHandler handles incoming requests from the peer.
// Return the result value (to be JSON-marshaled) or an error.
type RequestHandler func(method string, params json.RawMessage) (any, error)

// NotificationHandler handles incoming notifications (no response).
type NotificationHandler func(method string, params json.RawMessage)

// Session is a bidirectional JSON-RPC 2.0 session over an io.ReadWriter (a TCP conn).
type Session struct {
	rw        io.ReadWriter
	reader    *bufio.Reader
	writeMu   sync.Mutex // serializes writes to the underlying writer
	nextID    atomic.Int64
	pending   sync.Map // map[string]chan *Response
	onRequest RequestHandler
	onNotify  NotificationHandler
	onClose   func()
	closeOnce sync.Once
	closed    atomic.Bool
}

func NewSession(rw io.ReadWriter) *Session {
	s := &Session{rw: rw, reader: bufio.NewReader(rw)}
	return s
}

func (s *Session) HandleRequests(h RequestHandler)      { s.onRequest = h }
func (s *Session) HandleNotifications(h NotificationHandler) { s.onNotify = h }
func (s *Session) OnClosed(f func())                    { s.onClose = f }

// Run drives the read-loop until the connection closes or an unrecoverable
// error occurs. It is safe to call concurrently with Request/Notify.
func (s *Session) Run() error {
	defer s.Close()
	for {
		body, err := ReadOne(s.reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		s.dispatch(body)
	}
}

// Request sends a request and waits for the matching response.
func (s *Session) Request(method string, params any) (json.RawMessage, error) {
	if s.closed.Load() {
		return nil, errors.New("session closed")
	}
	id := s.nextID.Add(1)
	idJSON, _ := json.Marshal(id)
	var paramsJSON json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
		paramsJSON = b
	}
	req := Request{Jsonrpc: "2.0", ID: idJSON, Method: method, Params: paramsJSON}

	ch := make(chan *Response, 1)
	s.pending.Store(string(idJSON), ch)
	defer s.pending.Delete(string(idJSON))

	if err := s.write(req); err != nil {
		return nil, err
	}

	resp, ok := <-ch
	if !ok {
		return nil, errors.New("session closed before response")
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	return resp.Result, nil
}

// Notify sends a notification (no id, no response expected).
func (s *Session) Notify(method string, params any) error {
	if s.closed.Load() {
		return nil
	}
	var paramsJSON json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return err
		}
		paramsJSON = b
	}
	return s.write(Notification{Jsonrpc: "2.0", Method: method, Params: paramsJSON})
}

// Close shuts down the session. Safe to call multiple times.
func (s *Session) Close() {
	s.closeOnce.Do(func() {
		s.closed.Store(true)
		s.pending.Range(func(key, value any) bool {
			if ch, ok := value.(chan *Response); ok {
				close(ch)
			}
			s.pending.Delete(key)
			return true
		})
		if closer, ok := s.rw.(io.Closer); ok {
			_ = closer.Close()
		}
		if s.onClose != nil {
			s.onClose()
		}
	})
}

func (s *Session) write(msg any) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return Encode(s.rw, msg)
}

func (s *Session) dispatch(body []byte) {
	// Peek at the shape: request has "id" and "method", response has "id" but no "method",
	// notification has "method" but no "id".
	var peek struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
	}
	if err := json.Unmarshal(body, &peek); err != nil {
		return
	}
	hasID := len(peek.ID) > 0 && string(peek.ID) != "null"
	switch {
	case hasID && peek.Method != "":
		var req Request
		if err := json.Unmarshal(body, &req); err != nil {
			return
		}
		s.handleIncomingRequest(req)
	case hasID:
		var resp Response
		if err := json.Unmarshal(body, &resp); err != nil {
			return
		}
		s.handleIncomingResponse(resp)
	case peek.Method != "":
		var n Notification
		if err := json.Unmarshal(body, &n); err != nil {
			return
		}
		if s.onNotify != nil {
			s.onNotify(n.Method, n.Params)
		}
	}
}

func (s *Session) handleIncomingRequest(req Request) {
	go func() {
		var result any
		var rpcErr *RpcError
		if s.onRequest == nil {
			rpcErr = &RpcError{Code: CodeMethodNotFound, Message: "no handler"}
		} else {
			r, err := s.onRequest(req.Method, req.Params)
			if err != nil {
				rpcErr = &RpcError{Code: CodeServerError, Message: err.Error()}
			} else {
				result = r
			}
		}
		resp := Response{Jsonrpc: "2.0", ID: req.ID}
		if rpcErr != nil {
			resp.Error = rpcErr
		} else if result != nil {
			b, err := json.Marshal(result)
			if err != nil {
				resp.Error = &RpcError{Code: CodeInternalError, Message: err.Error()}
			} else {
				resp.Result = b
			}
		} else {
			resp.Result = json.RawMessage("null")
		}
		_ = s.write(resp)
	}()
}

func (s *Session) handleIncomingResponse(resp Response) {
	key := string(resp.ID)
	chAny, ok := s.pending.LoadAndDelete(key)
	if !ok {
		return
	}
	ch, _ := chAny.(chan *Response)
	ch <- &resp
}
