package commands

import (
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"github.com/zweiundeins/lith/writer"

	"starbase/internal/model"
)

var (
	tabIDRe = regexp.MustCompile(`^[a-z0-9]{8,32}$`)
	opIDRe  = regexp.MustCompile(`^[a-z0-9]{1,32}$`)
)

func validTab(sid, tabID string) error {
	if sid == "" || !tabIDRe.MatchString(tabID) {
		return errors.New("bad tab")
	}
	return nil
}

// validOp accepts no op (a client without op tracking) or an op id.
func validOp(op string) error {
	if op != "" && !opIDRe.MatchString(op) {
		return errors.New("bad op id")
	}
	return nil
}

// The keys under which the tab's row acks ops (TabState.Ops).
const (
	OpBrowse = "browse"
	OpSort   = "sort"
	OpShare  = "share"
)

// OpFlight is the key of a flight plan field's ops.
func OpFlight(field string) string { return "flight:" + field }

// ack records op as the newest op of key in the tab's row.
func ack(st *model.TabState, key, op string) {
	if op == "" {
		return
	}
	if st.Ops == nil {
		st.Ops = map[string]string{}
	}
	st.Ops[key] = op
}

// updateTab reads the tab's state (or seed when absent), applies fn and
// writes it back.
func updateTab(tx *writer.Tx, sid, tabID string, seed model.TabState, fn func(*model.TabState)) error {
	st := seed
	var raw string
	err := tx.QueryRow(`SELECT data FROM tab_state WHERE sid = ? AND tab_id = ?`, sid, tabID).Scan(&raw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return err
	default:
		if err := json.Unmarshal([]byte(raw), &st); err != nil {
			st = seed
		}
	}
	fn(&st)
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
		INSERT INTO tab_state (sid, tab_id, data, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT (sid, tab_id) DO UPDATE SET data = excluded.data, updated_at = excluded.updated_at`,
		sid, tabID, string(b), time.Now().Unix())
	return err
}

// SetBrowseFilter stores the gallery's search, category and sort for a tab.
type SetBrowseFilter struct {
	SID, TabID, Op string
	Browse         model.Browse
}

func (c SetBrowseFilter) Validate() error {
	return errors.Join(validTab(c.SID, c.TabID), validOp(c.Op))
}

func (c SetBrowseFilter) Apply(tx *writer.Tx) error {
	tx.Touch(TabTopic(c.TabID))
	return updateTab(tx, c.SID, c.TabID, model.TabState{}, func(st *model.TabState) {
		st.Browse = c.Browse.Normalize()
		ack(st, OpBrowse, c.Op)
	})
}

// TabKey names one tab's row.
type TabKey struct{ SID, Tab string }

// KeepTabs marks the rows of tabs whose page is open as touched, so the
// sweep keeps them. It touches no topic: nothing on those pages changes.
type KeepTabs struct{ Tabs []TabKey }

func (c KeepTabs) Apply(tx *writer.Tx) error {
	now := time.Now().Unix()
	for _, k := range c.Tabs {
		if _, err := tx.Exec(`UPDATE tab_state SET updated_at = ? WHERE sid = ? AND tab_id = ?`, now, k.SID, k.Tab); err != nil {
			return err
		}
	}
	return nil
}

// TabTTL is how long a tab's state outlives its last command, or its page
// (KeepTabs).
const TabTTL = 24 * time.Hour

// PruneTabs forgets tab state that has not been touched for a while. It
// touches no topic: a page open that long renders its defaults at its next
// frame.
type PruneTabs struct{ OlderThan time.Duration }

func (c PruneTabs) Apply(tx *writer.Tx) error {
	_, err := tx.Exec(`DELETE FROM tab_state WHERE updated_at < ?`, time.Now().Add(-c.OlderThan).Unix())
	return err
}
