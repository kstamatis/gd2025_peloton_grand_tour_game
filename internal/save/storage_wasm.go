//go:build js && wasm

package save

import (
	"fmt"
	"syscall/js"
)

// WASMStorage persists save slots in browser window.localStorage.
type WASMStorage struct {
	appID        string
	localStorage js.Value
}

// NewStorage creates a WASMStorage backed by window.localStorage.
func NewStorage(appID string) (Storage, error) {
	if appID == "" {
		return nil, fmt.Errorf("save: appID must not be empty")
	}
	win := js.Global().Get("window")
	if win.IsUndefined() || win.IsNull() {
		return nil, fmt.Errorf("save: browser window object is unavailable in WASM")
	}
	ls := win.Get("localStorage")
	if ls.IsUndefined() || ls.IsNull() {
		return nil, fmt.Errorf("save: window.localStorage is unavailable")
	}
	return &WASMStorage{appID: appID, localStorage: ls}, nil
}

func (ws *WASMStorage) key(slot string) string {
	return ws.appID + ":" + slot
}

// Save stores the serialized JSON payload in window.localStorage.
func (ws *WASMStorage) Save(slot string, data GameSaveData) error {
	if slot == "" {
		return fmt.Errorf("save: slot name must not be empty")
	}
	raw, err := MarshalSave(data)
	if err != nil {
		return err
	}
	ws.localStorage.Call("setItem", ws.key(slot), string(raw))
	return nil
}

// Load retrieves and validates the JSON payload from window.localStorage.
func (ws *WASMStorage) Load(slot string) (GameSaveData, error) {
	if slot == "" {
		return GameSaveData{}, fmt.Errorf("save: slot name must not be empty")
	}
	val := ws.localStorage.Call("getItem", ws.key(slot))
	if val.IsNull() || val.IsUndefined() {
		return GameSaveData{}, fmt.Errorf("save: slot %q not found in localStorage", ws.key(slot))
	}
	return UnmarshalSave([]byte(val.String()))
}
