// Package cqrs implements the write side of the application.
//
// Commands are values that mutate the database. HTTP handlers validate a
// command and enqueue it on the Bus, then return 204 immediately. A single
// writer goroutine drains the queue, applies every queued command inside one
// transaction (each in its own savepoint), commits, and then wakes the render
// streams affected by the batch through the Hub. Streams re-run their queries
// and push a fresh view. Commands never render HTML.
package cqrs

import (
	"context"
	"database/sql"
	"errors"
)

// Command mutates state. Apply runs inside the batch transaction.
type Command interface {
	Apply(ctx context.Context, tx *sql.Tx) error
}

// Scoped commands only affect views of one browser session. Commands that do
// not implement Scoped wake every stream (e.g. a star count changed).
type Scoped interface {
	Scope() (sid string)
}

// ErrInvalid marks a validation failure; handlers map it to 400.
var ErrInvalid = errors.New("invalid command")

// Validator is implemented by commands that can be checked before queueing.
type Validator interface {
	Validate() error
}
