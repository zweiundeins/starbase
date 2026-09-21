package cqrs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
)

// Bus is the single writer. Send enqueues without waiting; Exec waits for the
// batch containing the command to commit.
type Bus struct {
	db       *sql.DB
	hub      *Hub
	log      *slog.Logger
	q        chan envelope
	maxBatch int
}

type envelope struct {
	cmd  Command
	done chan error // nil for fire-and-forget
}

func NewBus(db *sql.DB, hub *Hub, log *slog.Logger) *Bus {
	return &Bus{db: db, hub: hub, log: log, q: make(chan envelope, 4096), maxBatch: 512}
}

var ErrQueueFull = errors.New("command queue full")

// Send enqueues cmd and returns immediately.
func (b *Bus) Send(cmd Command) error {
	if err := validate(cmd); err != nil {
		return err
	}
	select {
	case b.q <- envelope{cmd: cmd}:
		return nil
	default:
		return ErrQueueFull
	}
}

// Exec enqueues cmd and blocks until it has been applied and committed.
func (b *Bus) Exec(ctx context.Context, cmd Command) error {
	if err := validate(cmd); err != nil {
		return err
	}
	done := make(chan error, 1)
	select {
	case b.q <- envelope{cmd: cmd, done: done}:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func validate(cmd Command) error {
	if v, ok := cmd.(Validator); ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalid, err)
		}
	}
	return nil
}

// Run processes batches until ctx is cancelled.
func (b *Bus) Run(ctx context.Context) {
	batch := make([]envelope, 0, b.maxBatch)
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-b.q:
			batch = append(batch[:0], e)
		}
		// Drain whatever else is already queued: under load this groups many
		// commands into one transaction and one render per stream.
	drain:
		for len(batch) < b.maxBatch {
			select {
			case e := <-b.q:
				batch = append(batch, e)
			default:
				break drain
			}
		}
		b.apply(ctx, batch)
	}
}

func (b *Bus) apply(ctx context.Context, batch []envelope) {
	results := make([]error, len(batch))
	all := false
	sids := map[string]struct{}{}

	err := func() error {
		tx, err := b.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		for i, e := range batch {
			// A savepoint per command: one failing command does not
			// abort the rest of the batch.
			if _, err := tx.ExecContext(ctx, "SAVEPOINT cmd"); err != nil {
				return err
			}
			if err := e.cmd.Apply(ctx, tx); err != nil {
				results[i] = err
				if _, err := tx.ExecContext(ctx, "ROLLBACK TO cmd"); err != nil {
					return err
				}
			}
			if _, err := tx.ExecContext(ctx, "RELEASE cmd"); err != nil {
				return err
			}
			if results[i] != nil {
				continue
			}
			if s, ok := e.cmd.(Scoped); ok {
				sids[s.Scope()] = struct{}{}
			} else {
				all = true
			}
		}
		return tx.Commit()
	}()

	for i, e := range batch {
		res := results[i]
		if err != nil {
			res = err
		}
		if res != nil {
			b.log.Warn("command failed", "cmd", fmt.Sprintf("%T", e.cmd), "err", res)
		}
		if e.done != nil {
			e.done <- res
		}
	}
	if err == nil && (all || len(sids) > 0) {
		b.hub.Notify(all, sids)
	}
}
