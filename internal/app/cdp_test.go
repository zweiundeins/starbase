package app_test

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// cdp is a Chrome DevTools Protocol client over --remote-debugging-pipe
// (fd 3 in, fd 4 out, messages ended by NUL), attached to one page.
type cdp struct {
	mu      sync.Mutex
	w       io.Writer
	id      int
	waits   map[int]chan cdpReply
	session string
}

type cdpReply struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *cdp) call(session, method string, params any) (json.RawMessage, error) {
	c.mu.Lock()
	c.id++
	id := c.id
	ch := make(chan cdpReply, 1)
	c.waits[id] = ch
	msg := map[string]any{"id": id, "method": method, "params": params}
	if session != "" {
		msg["sessionId"] = session
	}
	b, _ := json.Marshal(msg)
	_, err := c.w.Write(append(b, 0))
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	select {
	case r := <-ch:
		if r.Error != nil {
			return nil, fmt.Errorf("%s: %s", method, r.Error.Message)
		}
		return r.Result, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("%s: no answer", method)
	}
}

func (c *cdp) read(r io.Reader) {
	br := bufio.NewReader(r)
	for {
		b, err := br.ReadBytes(0)
		if err != nil {
			return
		}
		var m struct {
			ID int `json:"id"`
			cdpReply
		}
		if json.Unmarshal(b[:len(b)-1], &m) == nil && m.ID != 0 {
			c.mu.Lock()
			ch := c.waits[m.ID]
			delete(c.waits, m.ID)
			c.mu.Unlock()
			if ch != nil {
				ch <- m.cdpReply
			}
		}
	}
}

// probeCDP is probeBundle in a Chrome the page can drive through POST
// /__probe/cdp: a JSON list of {method, params} run in order on the page's
// session (real input, emulated media, the accessibility tree), answered
// with their results. {method: "shot", params: {name, clip}} saves a PNG
// to $STARBASE_SHOTS/<name>.png when that is set. more adds test handlers by path.
func probeCDP(t *testing.T, path, script string, more ...map[string]http.HandlerFunc) []byte {
	t.Helper()
	var c *cdp
	var ready = make(chan struct{})
	extra := map[string]http.HandlerFunc{}
	for _, m := range more {
		maps.Copy(extra, m)
	}
	extra["/__probe/cdp"] = func(w http.ResponseWriter, r *http.Request) {
		<-ready
		var cmds []struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&cmds); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		out := []json.RawMessage{}
		for _, cmd := range cmds {
			if cmd.Method == "shot" {
				res, err := c.shot(cmd.Params)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				out = append(out, res)
				continue
			}
			res, err := c.call(c.session, cmd.Method, cmd.Params)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			out = append(out, res)
		}
		json.NewEncoder(w).Encode(out)
	}
	chrome := findChrome(t)
	start := func(url string, results <-chan []byte) []byte {
		args := []string{"--headless=new", "--disable-gpu", "--window-size=1440,1000", "--remote-debugging-pipe", "about:blank"}
		if os.Geteuid() == 0 || os.Getenv("STARBASE_CHROME_NO_SANDBOX") == "1" {
			args = append([]string{"--no-sandbox"}, args...)
		}
		cmd := exec.Command(chrome, args...)
		inR, inW, _ := os.Pipe()
		outR, outW, _ := os.Pipe()
		cmd.ExtraFiles = []*os.File{inR, outW}
		var stderr strings.Builder
		cmd.Stdout, cmd.Stderr = io.Discard, &stderr
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		inR.Close()
		outW.Close()
		defer func() { cmd.Process.Kill(); cmd.Wait(); inW.Close(); outR.Close() }()
		c = &cdp{w: inW, waits: map[int]chan cdpReply{}}
		go c.read(outR)
		res, err := c.call("", "Target.getTargets", map[string]any{})
		if err != nil {
			if strings.Contains(stderr.String(), "No usable sandbox") {
				t.Skip("Chrome has no usable sandbox here; set STARBASE_CHROME_NO_SANDBOX=1 to run this test")
			}
			t.Fatalf("%v\n%s", err, stderr.String())
		}
		var targets struct {
			TargetInfos []struct{ TargetID, Type string } `json:"targetInfos"`
		}
		json.Unmarshal(res, &targets)
		var page string
		for _, ti := range targets.TargetInfos {
			if ti.Type == "page" {
				page = ti.TargetID
			}
		}
		res, err = c.call("", "Target.attachToTarget", map[string]any{"targetId": page, "flatten": true})
		if err != nil {
			t.Fatal(err)
		}
		var att struct{ SessionID string }
		json.Unmarshal(res, &att)
		c.session = att.SessionID
		close(ready)
		if _, err := c.call(c.session, "Page.navigate", map[string]any{"url": url}); err != nil {
			t.Fatal(err)
		}
		select {
		case b := <-results:
			return b
		case <-time.After(120 * time.Second):
			t.Fatalf("the page posted nothing\n%s", stderr.String())
		}
		return nil
	}
	return probeBundleRun(t, path, script, extra, start)
}

// shot saves a screenshot of the clip ({x, y, width, height} in CSS px).
func (c *cdp) shot(raw json.RawMessage) (json.RawMessage, error) {
	var p struct {
		Name string         `json:"name"`
		Clip map[string]any `json:"clip"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	dir := os.Getenv("STARBASE_SHOTS")
	if dir == "" {
		return json.RawMessage(`null`), nil
	}
	if strings.ContainsAny(p.Name, `/\`) || p.Name == "" {
		return nil, errors.New("bad shot name")
	}
	params := map[string]any{"format": "png", "captureBeyondViewport": true}
	if p.Clip != nil {
		p.Clip["scale"] = 1
		params["clip"] = p.Clip
	}
	res, err := c.call(c.session, "Page.captureScreenshot", params)
	if err != nil {
		return nil, err
	}
	var img struct{ Data string }
	json.Unmarshal(res, &img)
	b, err := base64.StdEncoding.DecodeString(img.Data)
	if err != nil {
		return nil, err
	}
	file := filepath.Join(dir, p.Name+".png")
	if err := os.WriteFile(file, b, 0o644); err != nil {
		return nil, err
	}
	return json.Marshal(file)
}
