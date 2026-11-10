package commands

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/zweiundeins/lith/writer"

	"starbase/internal/model"
)

// SetFlight sets one field of the Showcase's flight plan (the commands demo).
// The server validates and normalizes; the page re-renders from the stored
// plan, which is what confirms, corrects or refuses the control that sent it.
type SetFlight struct {
	SID, TabID  string
	Name, Value string
}

var callsignRe = regexp.MustCompile(`^[A-Z0-9-]{2,16}$`)

// The refusals a person can cause with the controls, which the tab's row
// records and the page shows.
const (
	RefusedThrust   = "Thrust above 90% needs the flight director's approval. The slider is back at the stored thrust."
	RefusedCallsign = "A call sign is 2 to 16 letters, digits or dashes. The field is back at the stored one."
)

// Validate refuses what the controls never send: an unknown field, a thrust
// off the slider, shields that are not on or off.
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
	case "shields":
		if c.Value != "true" && c.Value != "false" {
			return errors.New("shields are true or false")
		}
	case "callsign":
	default:
		return errors.New("unknown field")
	}
	return nil
}

// Apply stores the field, or records why not: a thrust above 90 and a
// malformed call sign are refusals in the tab's row. Either way the field's
// rev counts the command, so its control takes the server's value.
func (c SetFlight) Apply(tx *writer.Tx) error {
	tx.Touch(TabTopic(c.TabID))
	return updateTab(tx, c.SID, c.TabID, model.TabState{}, func(st *model.TabState) {
		f := st.Flight.OrDefault()
		f.Refusal = ""
		switch c.Name {
		case "thrust":
			f.ThrustRev++
			if n, _ := strconv.Atoi(c.Value); n > 90 {
				f.Refusal = RefusedThrust
			} else {
				f.Thrust = n
			}
		case "shields":
			f.Shields = c.Value == "true"
		case "callsign":
			f.CallsignRev++
			if sign := strings.ToUpper(strings.TrimSpace(c.Value)); callsignRe.MatchString(sign) {
				f.Callsign = sign // the server normalizes
			} else {
				f.Refusal = RefusedCallsign
			}
		}
		st.Flight = f
	})
}
