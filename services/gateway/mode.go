//go:build !e2e

package main

// e2e reports whether this gateway is built for the end-to-end tests, without its locks
// (decision 107). Only `go build -tags e2e` builds it so.
const e2e = false
