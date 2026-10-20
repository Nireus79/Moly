package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Native messaging host. The browser starts the Moly binary with the extension's origin as first argument and talks to
// it over stdin/stdout in frames (4-byte length in native byte order, then UTF-8 JSON). The host starts the backend if
// it is not running, hands the extension the port and the session token, and keeps the backend alive with heartbeats
// until the browser closes the pipe. Nothing but frames may be written to stdout.

// version is set at build time (-ldflags "-X main.version=..."); "dev" for a source build.
var version = "dev"

const (
	hostName          = "com.moly.backend_host"
	hostVersion       = "1"
	maxFrame          = 1 << 20 // the browser limits host-to-browser messages to 1 MB
	backendStartLimit = 90 * time.Second
	heartbeatEvery    = 10 * time.Second
)

// runSubcommand handles the invocations that do not start the server.
func runSubcommand(args []string) (handled bool, code int) {
	if len(args) == 0 {
		return false, 0
	}
	switch first := args[0]; {
	case strings.HasPrefix(first, "chrome-extension://"):
		return true, runNativeHost(first, os.Stdin, os.Stdout)
	case len(args) >= 2 && strings.HasSuffix(first, ".json") && isGeckoID(args[1]):
		// Firefox starts the host with the manifest path and the add-on ID.
		return true, runNativeHost(args[1], os.Stdin, os.Stdout)
	case first == "--register-host":
		return true, runRegisterHost(args[1:])
	case first == "--unregister-host":
		return true, runUnregisterHost()
	case first == "--version":
		fmt.Printf("moly %s (host protocol %s)\n", version, hostVersion)
		return true, 0
	case first == "--stop":
		return true, runStop(os.Stdout)
	case first == "--doctor":
		return true, runDoctor(os.Stdout)
	}
	return false, 0
}

func readFrame(r io.Reader) ([]byte, error) {
	var n uint32
	if err := binary.Read(r, binary.NativeEndian, &n); err != nil {
		return nil, err
	}
	if n > maxFrame {
		return nil, fmt.Errorf("frame of %d bytes is over the %d byte limit", n, maxFrame)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func writeFrame(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(data) > maxFrame {
		return fmt.Errorf("reply of %d bytes is over the %d byte limit", len(data), maxFrame)
	}
	var out bytes.Buffer
	binary.Write(&out, binary.NativeEndian, uint32(len(data)))
	out.Write(data)
	_, err = w.Write(out.Bytes()) // one write per frame
	return err
}

type nativeHost struct {
	base string // http://127.0.0.1:<port>
	http *http.Client
	out  io.Writer
	outM sync.Mutex

	startM   sync.Mutex // one start at a time
	hostID   string     // attachment at the backend
	attached sync.Mutex
	stop     chan struct{}

	ollama string
	spawn  func() error // starts the backend; replaced in tests
}

func newNativeHost(port, ollama string, out io.Writer) *nativeHost {
	h := &nativeHost{
		base:   "http://127.0.0.1" + port,
		http:   &http.Client{Timeout: 3 * time.Second},
		out:    out,
		stop:   make(chan struct{}),
		ollama: ollama,
	}
	h.spawn = h.spawnBackend
	return h
}

func (h *nativeHost) send(v any) {
	h.outM.Lock()
	defer h.outM.Unlock()
	if err := writeFrame(h.out, v); err != nil {
		log.Printf("[host] cannot write a frame: %v", err)
	}
}

// health returns the HTTP status of /api/status and the version, or 0 when nothing answers.
func (h *nativeHost) health() (int, string) {
	resp, err := h.http.Get(h.base + "/api/status")
	if err != nil {
		return 0, ""
	}
	defer resp.Body.Close()
	var body struct {
		Version string `json:"version"`
	}
	json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&body)
	return resp.StatusCode, body.Version
}

func (h *nativeHost) spawnBackend() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	dir := molyDataDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	logPath := filepath.Join(dir, "moly.log")
	if st, err := os.Stat(logPath); err == nil && st.Size() > 5<<20 {
		os.Remove(logPath) // keep the log small
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer logFile.Close()
	cmd := exec.Command(exe)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "MOLY_PRODUCTION=1", "MOLY_IDLE_EXIT=1")
	cmd.Stdout, cmd.Stderr = logFile, logFile
	detachProcess(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// ensureBackend returns when the backend answers 200 (or 423, locked), starting it once if nothing listens.
func (h *nativeHost) ensureBackend() (version string, err error) {
	h.startM.Lock()
	defer h.startM.Unlock()
	spawned := false
	deadline := time.Now().Add(backendStartLimit)
	for time.Now().Before(deadline) {
		code, ver := h.health()
		switch {
		case code == http.StatusOK, code == http.StatusLocked:
			// 423: the backend is up and waits for the person (passphrase, or where to keep the database key).
			return ver, nil
		case code == 0 && !spawned:
			if err := h.spawn(); err != nil {
				return "", fmt.Errorf("cannot start the backend: %w", err)
			}
			spawned = true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return "", errors.New("the backend did not become ready in time (see ~/.moly/moly.log)")
}

func (h *nativeHost) token() (string, error) {
	data, err := os.ReadFile(filepath.Join(molyDataDir(), tokenFile))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (h *nativeHost) post(path, token string) (int, []byte, error) {
	req, _ := http.NewRequest(http.MethodPost, h.base+path, nil)
	req.Header.Set(tokenHeader, token)
	resp, err := h.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, body, nil
}

// attach registers this host at the backend and starts the heartbeat (once).
func (h *nativeHost) attach(token string) error {
	h.attached.Lock()
	defer h.attached.Unlock()
	if h.hostID != "" {
		return nil
	}
	code, body, err := h.post("/internal/attach", token)
	if err != nil || code != http.StatusOK {
		return fmt.Errorf("cannot attach to the backend (status %d): %v", code, err)
	}
	var r struct {
		HostID string `json:"hostId"`
	}
	if err := json.Unmarshal(body, &r); err != nil || r.HostID == "" {
		return errors.New("the backend gave no host id")
	}
	h.hostID = r.HostID
	go h.heartbeats()
	return nil
}

func (h *nativeHost) heartbeats() {
	tick := time.NewTicker(heartbeatEvery)
	defer tick.Stop()
	for {
		select {
		case <-h.stop:
			return
		case <-tick.C:
			tok, err := h.token()
			if err != nil {
				continue
			}
			code, _, err := h.post("/internal/heartbeat?id="+h.hostID, tok)
			if err != nil || code == http.StatusNotFound || code == http.StatusUnauthorized {
				// The backend restarted or went away: tell the extension, which asks for "start" again.
				h.attached.Lock()
				h.hostID = ""
				h.attached.Unlock()
				h.send(map[string]any{"event": "backend-down"})
				return
			}
		}
	}
}

func (h *nativeHost) detach() {
	close(h.stop)
	h.attached.Lock()
	id := h.hostID
	h.attached.Unlock()
	if id == "" {
		return
	}
	if tok, err := h.token(); err == nil {
		h.post("/internal/detach?id="+id, tok)
	}
}

type hostRequest struct {
	Action string `json:"action"`
	ID     any    `json:"id,omitempty"` // echoed back, so the extension can match replies
}

func (h *nativeHost) handle(raw []byte) {
	var req hostRequest
	reply := func(m map[string]any) {
		if req.ID != nil {
			m["id"] = req.ID
		}
		h.send(m)
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		reply(map[string]any{"success": false, "error": "invalid message"})
		return
	}
	switch req.Action {
	case "ping":
		reply(map[string]any{"success": true, "pong": true, "hostVersion": hostVersion})
	case "start", "start-backend":
		ver, err := h.ensureBackend()
		if err != nil {
			reply(map[string]any{"success": false, "error": err.Error()})
			return
		}
		tok, err := h.token()
		if err != nil {
			reply(map[string]any{"success": false, "error": "no session token: " + err.Error()})
			return
		}
		if err := h.attach(tok); err != nil {
			reply(map[string]any{"success": false, "error": err.Error()})
			return
		}
		reply(map[string]any{"success": true, "url": h.base, "token": tok, "version": ver})
	case "status":
		code, ver := h.health()
		state := map[int]string{http.StatusOK: "running", http.StatusServiceUnavailable: "starting", http.StatusLocked: "locked"}[code]
		if state == "" {
			state = "stopped"
		}
		reply(map[string]any{"success": true, "state": state, "running": code == http.StatusOK || code == http.StatusLocked, "version": ver})
	case "models", "get-models":
		models, err := h.ollamaModels()
		if err != nil {
			reply(map[string]any{"success": true, "models": []any{}, "note": err.Error()})
			return
		}
		reply(map[string]any{"success": true, "models": models})
	default:
		reply(map[string]any{"success": false, "error": "unknown action: " + req.Action})
	}
}

func (h *nativeHost) ollamaModels() ([]map[string]string, error) {
	resp, err := h.http.Get(strings.TrimRight(h.ollama, "/") + "/api/tags")
	if err != nil {
		return nil, errors.New("Ollama is not running")
	}
	defer resp.Body.Close()
	var body struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return nil, err
	}
	out := []map[string]string{}
	for _, m := range body.Models {
		out = append(out, map[string]string{"name": m.Name})
	}
	return out, nil
}

// runNativeHost serves the browser until it closes the pipe.
func runNativeHost(origin string, in io.Reader, out io.Writer) int {
	log.SetOutput(os.Stderr) // stdout carries frames only
	if ids := allowedExtensionIDs(); len(ids) > 0 {
		ok := false
		for _, id := range ids {
			if o := strings.TrimSuffix(origin, "/"); o == "chrome-extension://"+id || (isGeckoID(id) && o == id) {
				ok = true
			}
		}
		if !ok {
			log.Printf("[host] refusing origin %s", origin)
			return 1
		}
	}
	cfg := LoadConfig()
	h := newNativeHost(cfg.Port, cfg.OllamaEndpoint, out)
	defer h.detach()
	for {
		raw, err := readFrame(in)
		if err != nil {
			if err != io.EOF && !errors.Is(err, io.ErrUnexpectedEOF) {
				log.Printf("[host] %v", err)
			}
			return 0
		}
		go h.handle(raw)
	}
}
