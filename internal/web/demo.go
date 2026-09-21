package web

import (
	"math"
	"net/http"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

// Telemetry is one sample of the demo mission.
type Telemetry struct {
	T     float64 `json:"t"`     // seconds since liftoff (negative: countdown)
	Stage string  `json:"stage"` // PAD, ASCENT, MECO, ORBIT
	Alt   float64 `json:"alt"`   // km
	Vel   float64 `json:"vel"`   // km/s
	Fuel  float64 `json:"fuel"`  // %
	Temp  float64 `json:"temp"`  // °C, hull
	Pitch float64 `json:"pitch"` // degrees above the horizon
}

// missionCycle is the length of one simulated flight.
const missionCycle = 90 * time.Second

// TelemetryAt is a pure function of wall-clock time, so the stream holds no
// state and every viewer watches the same flight.
func TelemetryAt(now time.Time) Telemetry {
	cycle := float64(now.UnixMilli()%missionCycle.Milliseconds()) / 1000
	const countdown, burn = 8.0, 58.0
	wobble := math.Sin(cycle*2.3)*0.5 + math.Sin(cycle*5.1)*0.25
	t := cycle - countdown
	var m Telemetry
	switch {
	case t < 0:
		m = Telemetry{Stage: "PAD", Fuel: 100, Temp: 18 + wobble, Pitch: 90}
	case t < burn:
		p := t / burn
		m = Telemetry{
			Stage: "ASCENT",
			Alt:   410 * math.Pow(p, 1.7),
			Vel:   7.8 * math.Pow(p, 1.25),
			Fuel:  100 - 96*p,
			Temp:  18 + 1150*math.Sin(math.Pi*math.Min(1, p*1.35))*0.9 + wobble*6,
			Pitch: 90 * (1 - math.Pow(p, 0.8)),
		}
		if p > 0.97 {
			m.Stage = "MECO"
		}
	default:
		m = Telemetry{Stage: "ORBIT", Alt: 410 + wobble*2, Vel: 7.8 + wobble*0.01, Fuel: 4, Temp: math.Max(-60, 180-(t-burn)*14) + wobble, Pitch: 0}
	}
	m.T = math.Round(t*10) / 10
	round := func(v float64) float64 { return math.Round(v*10) / 10 }
	m.Alt, m.Vel, m.Fuel, m.Temp, m.Pitch = round(m.Alt), math.Round(m.Vel*100)/100, round(m.Fuel), round(m.Temp), round(m.Pitch)
	return m
}

// demoTelemetry streams the demo mission as signal patches ($_tm): a pure
// query stream, no commands, no database.
func (s *Server) demoTelemetry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*") // public; used from the playground sandbox
	sse := datastar.NewSSE(w, r)
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		if err := sse.MarshalAndPatchSignals(map[string]any{"_tm": TelemetryAt(time.Now())}); err != nil {
			return
		}
		select {
		case <-tick.C:
		case <-r.Context().Done():
			return
		case <-s.ctx.Done():
			return
		}
	}
}
