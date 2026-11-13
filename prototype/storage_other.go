//go:build !(js && wasm)

package main

import "errors"

var errNoIDB = errors.New("IndexedDB exists only in the browser build")

func storeGet(string) (string, error) { return "", errNoIDB }
func storePut(string, string) error   { return errNoIDB }
