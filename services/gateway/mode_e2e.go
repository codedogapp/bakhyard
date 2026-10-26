//go:build e2e

package main

// e2e reports whether this gateway is built for the end-to-end tests, without its locks
// (decision 107).
const e2e = true
