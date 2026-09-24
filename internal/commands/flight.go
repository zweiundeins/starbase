package commands

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"starbase/internal/model"
)

// SetFlight sets one field of the Showcase's flight plan (the commands demo).
// The server validates and normalizes; the page re-renders from the stored
// plan, which is what confirms (or corrects) the control that sent it.
type SetFlight struct {
	SID, TabID  string
	Name, Value string
}

var callsignRe = regexp.MustCompile(`^[A-Z0-9-]{2,16}$`)

// ErrThrust is the demo's business rule: a command the server rejects.
var ErrThrust = errors.New("thrust above 90% needs the flight director's approval")

func (c SetFlight) Validate() error {
	if err := validTab(c.SID, c.TabID); err != nil {
		return err
	}
	switch c.Name {
	case "thrust":
		n, err := strconv.Atoi(c.Value)
		if err != nil || n < 0 || n > 100 {
			return errors.New("thrust is 0 to 100")
		}
		if n > 90 {
			return ErrThrust
		}
	case "shields":
		if c.Value != "true" && c.Value != "false" {
			return errors.New("shields are true or false")
		}
	case "callsign":
		if !callsignRe.MatchString(strings.ToUpper(strings.TrimSpace(c.Value))) {
			return errors.New("a call sign is 2 to 16 letters, digits or dashes")
		}
	default:
		return errors.New("unknown field")
	}
	return nil
}

func (c SetFlight) Scope() string { return c.SID }

func (c SetFlight) Apply(ctx context.Context, tx *sql.Tx) error {
	return updateTab(ctx, tx, c.SID, c.TabID, model.TabState{}, func(st *model.TabState) {
		f := st.Flight.OrDefault()
		switch c.Name {
		case "thrust":
			f.Thrust, _ = strconv.Atoi(c.Value)
		case "shields":
			f.Shields = c.Value == "true"
		case "callsign":
			f.Callsign = strings.ToUpper(strings.TrimSpace(c.Value)) // the server normalizes
			f.CallsignRev++                                          // answered, even when nothing changed
		}
		st.Flight = f
	})
}
