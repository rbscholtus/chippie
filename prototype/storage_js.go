//go:build js && wasm

package main

import (
	"errors"
	"syscall/js"
)

// IndexedDB from Go: the callback API is wrapped in goroutines and channels.
// Callers never block the game loop; results arrive on a channel polled in Update.

const dbName, storeName = "chippie-spike", "kv"

func openDB() (js.Value, error) {
	idb := js.Global().Get("indexedDB")
	if idb.IsUndefined() || idb.IsNull() {
		return js.Value{}, errors.New("indexedDB not available")
	}
	req := idb.Call("open", dbName, 1)
	done := make(chan error, 1)
	var fns []js.Func
	on := func(name string, f func(js.Value)) {
		fn := js.FuncOf(func(this js.Value, args []js.Value) any { f(args[0]); return nil })
		fns = append(fns, fn)
		req.Set(name, fn)
	}
	on("onupgradeneeded", func(js.Value) { req.Get("result").Call("createObjectStore", storeName) })
	on("onsuccess", func(js.Value) { done <- nil })
	on("onerror", func(js.Value) { done <- errors.New(req.Get("error").Call("toString").String()) })
	err := <-done
	for _, fn := range fns {
		fn.Release()
	}
	if err != nil {
		return js.Value{}, err
	}
	return req.Get("result"), nil
}

func request(req js.Value) (js.Value, error) {
	done := make(chan error, 1)
	ok := js.FuncOf(func(js.Value, []js.Value) any { done <- nil; return nil })
	fail := js.FuncOf(func(js.Value, []js.Value) any {
		done <- errors.New(req.Get("error").Call("toString").String())
		return nil
	})
	defer ok.Release()
	defer fail.Release()
	req.Set("onsuccess", ok)
	req.Set("onerror", fail)
	if err := <-done; err != nil {
		return js.Value{}, err
	}
	return req.Get("result"), nil
}

func storeGet(key string) (string, error) {
	db, err := openDB()
	if err != nil {
		return "", err
	}
	defer db.Call("close")
	st := db.Call("transaction", storeName, "readonly").Call("objectStore", storeName)
	v, err := request(st.Call("get", key))
	if err != nil || v.IsUndefined() || v.IsNull() {
		return "", err
	}
	return v.String(), nil
}

func storePut(key, value string) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Call("close")
	st := db.Call("transaction", storeName, "readwrite").Call("objectStore", storeName)
	_, err = request(st.Call("put", value, key))
	return err
}
