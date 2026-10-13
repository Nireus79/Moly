package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"moly/keystore"
)

// Until the database key is available (the person has to unlock a passphrase or choose where the key lives), the backend
// runs only this gate: status, the keystore routes, host attach and shutdown. Everything else answers 503 "starting".

func registerKeyRoutes(mux *http.ServeMux, store *keystore.Store, version string) {
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		code, state := http.StatusServiceUnavailable, "starting"
		if st := store.Status(); st.State != keystore.StateReady {
			code, state = http.StatusLocked, "locked"
		}
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]string{"status": state, "version": version})
	})
	// Everything else waits for the key.
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusLocked, map[string]string{"error": "Moly is waiting for the database key."})
	})
	mux.HandleFunc("GET /api/keystore/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, store.Status())
	})
	mux.HandleFunc("POST /api/keystore/recheck", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, store.Recheck())
	})
	mux.HandleFunc("POST /api/keystore/unlock", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Passphrase string `json:"passphrase"`
		}
		if !readBody(w, r, &body) {
			return
		}
		switch err := store.Unlock(body.Passphrase); {
		case err == nil:
			writeJSON(w, http.StatusOK, store.Status())
		case errors.Is(err, keystore.ErrWrongPassphrase):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "That passphrase is not right."})
		case errors.Is(err, keystore.ErrNotNeeded):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Moly is not waiting for a passphrase."})
		default:
			log.Printf("[Moly] unlock: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "The key could not be opened."})
		}
	})
	mux.HandleFunc("POST /api/keystore/setup", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Mode            string `json:"mode"`
			Passphrase      string `json:"passphrase"`
			ConfirmInsecure bool   `json:"confirmInsecure"`
		}
		if !readBody(w, r, &body) {
			return
		}
		switch err := store.Setup(body.Mode, body.Passphrase, body.ConfirmInsecure); {
		case err == nil:
			writeJSON(w, http.StatusOK, store.Status())
		case errors.Is(err, keystore.ErrShortPassphrase), errors.Is(err, keystore.ErrNeedsConfirm), errors.Is(err, keystore.ErrNotNeeded):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		default:
			log.Printf("[Moly] key setup: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "The key could not be set up."})
		}
	})
}

// waitForKey blocks until the key is available. It returns false when Moly is told to stop (signal, shutdown request, or no
// browser for the idle period) before that.
func waitForKey(store *keystore.Store, hosts *hostRegistry, quit <-chan struct{}, idleExit bool) bool {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-store.Ready():
			return true
		case sig := <-stop:
			log.Printf("[Moly] %v received while waiting for the database key", sig)
			return false
		case <-quit:
			log.Printf("[Moly] stop requested while waiting for the database key")
			return false
		case <-tick.C:
			if idleExit && hosts.Idle() {
				log.Printf("[Moly] no browser attached for %v while waiting for the database key, exiting", idleGrace)
				return false
			}
		}
	}
}
