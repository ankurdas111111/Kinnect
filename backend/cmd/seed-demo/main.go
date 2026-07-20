// seed-demo populates a Kinnect database with a realistic demo family for the
// Ask-the-Map copilot: four users, contact links, saved places, seven days of
// GPS traces (school runs, office commutes, market trips), geofence events,
// and daily activity — all timed relative to NOW so "where is everyone right
// now?" always has a live answer.
//
//	DATABASE_URL=postgres://... go run ./cmd/seed-demo
//
// Idempotent: re-running wipes and re-creates only the demo users (matched by
// their @demo.kinnect emails).
//
// Demo login: demo@kinnect.app / demo1234
package main

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	"kinnect-v3/internal/auth"
	"kinnect-v3/internal/db"
)

// ─── Demo geography (Bengaluru) ──────────────────────────────────────────────

type place struct {
	name     string
	lat, lng float64
	radiusM  float64
	icon     string
}

var (
	home   = place{"Home", 12.93520, 77.62450, 150, "🏠"}
	school = place{"School", 12.98630, 77.60340, 200, "🏫"}
	office = place{"Office", 12.97160, 77.59460, 200, "🏢"}
	market = place{"Market", 12.92790, 77.62710, 120, "🛒"}
	park   = place{"Park", 12.93080, 77.61800, 150, "🌳"}
)

const (
	walkSpeed  = 1.4  // m/s
	driveSpeed = 9.0  // m/s (city traffic)
	demoPass   = "demo1234"
)

type member struct {
	first, last, email string
	id                 string
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fail("DATABASE_URL is not set")
	}
	pool, err := db.NewPool(dsn)
	if err != nil {
		fail("connect: %v", err)
	}
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := db.InitDB(ctx, pool.DB); err != nil {
		fail("init schema: %v", err)
	}

	rng := rand.New(rand.NewSource(42)) // deterministic traces

	members := []*member{
		{first: "Ankur", last: "Demo", email: "demo@kinnect.app"},
		{first: "Dad", last: "Demo", email: "dad@demo.kinnect"},
		{first: "Mom", last: "Demo", email: "mom@demo.kinnect"},
		{first: "Zara", last: "Demo", email: "zara@demo.kinnect"},
	}

	// ── Wipe previous demo data (cascades take positions/places/events) ──
	fmt.Println("wiping previous demo users...")
	if _, err := pool.DB.ExecContext(ctx,
		`DELETE FROM users WHERE email = 'demo@kinnect.app' OR email LIKE '%@demo.kinnect'`); err != nil {
		fail("wipe: %v", err)
	}

	// ── Users ──
	hash, err := auth.HashPassword(demoPass)
	if err != nil {
		fail("hash: %v", err)
	}
	now := time.Now()
	for i, m := range members {
		code := fmt.Sprintf("DM%04d", i+1)
		if err := pool.DB.QueryRowContext(ctx, `
			INSERT INTO users (first_name, last_name, password_hash, role, share_code, email, created_at)
			VALUES ($1, $2, $3, 'user', $4, $5, $6) RETURNING id`,
			m.first, m.last, hash, code, m.email, now.UnixMilli()).Scan(&m.id); err != nil {
			fail("insert user %s: %v", m.first, err)
		}
		fmt.Printf("user %-6s %s (share code %s)\n", m.first, m.id, code)
	}
	me := members[0]

	// ── Contacts: bidirectional rows (cache expects both directions) ──
	for _, other := range members[1:] {
		for _, pair := range [][2]string{{me.id, other.id}, {other.id, me.id}} {
			if _, err := pool.DB.ExecContext(ctx, `
				INSERT INTO contacts (owner_id, contact_id) VALUES ($1, $2)
				ON CONFLICT DO NOTHING`, pair[0], pair[1]); err != nil {
				fail("contact: %v", err)
			}
		}
	}

	// ── Saved places (owned by the demo login) ──
	for _, p := range []place{home, school, office, market, park} {
		if _, err := pool.DB.ExecContext(ctx, `
			INSERT INTO saved_places (user_id, name, icon, latitude, longitude, radius_m, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			me.id, p.name, p.icon, p.lat, p.lng, p.radiusM, now.UnixMilli()); err != nil {
			fail("place %s: %v", p.name, err)
		}
	}

	// ── Seven days of traces ──
	fmt.Println("generating traces...")
	gen := &generator{ctx: ctx, db: pool.DB, rng: rng}
	for dayOffset := -6; dayOffset <= 0; dayOffset++ {
		day := now.AddDate(0, 0, dayOffset)
		weekday := day.Weekday() != time.Saturday && day.Weekday() != time.Sunday

		// Zara: school days — home → school 7:55, school dwell, home 15:40, park some days.
		if weekday {
			gen.trip(members[3], day, 7, 55, home, school, driveSpeed)
			gen.dwell(members[3], day, 8, 25, 15, 30, school)
			gen.trip(members[3], day, 15, 32, school, home, driveSpeed)
			if dayOffset%2 == 0 {
				gen.trip(members[3], day, 17, 0, home, park, walkSpeed)
				gen.dwell(members[3], day, 17, 25, 18, 10, park)
				gen.trip(members[3], day, 18, 12, park, home, walkSpeed)
			} else {
				gen.dwell(members[3], day, 16, 0, 21, 30, home)
			}
		} else {
			gen.dwell(members[3], day, 9, 0, 11, 0, home)
			gen.trip(members[3], day, 11, 5, home, park, walkSpeed)
			gen.dwell(members[3], day, 11, 30, 12, 45, park)
			gen.trip(members[3], day, 12, 48, park, home, walkSpeed)
		}

		// Dad: office days with an unusual stop Wednesday; weekend errands.
		if weekday {
			gen.trip(members[1], day, 9, 5, home, office, driveSpeed)
			gen.dwell(members[1], day, 9, 40, 18, 15, office)
			if day.Weekday() == time.Wednesday {
				// The demo question: "where did Dad stop on his way home?"
				mid := place{"", (office.lat + home.lat) / 2, (office.lng + home.lng)/2 + 0.004, 0, ""}
				gen.trip(members[1], day, 18, 18, office, mid, driveSpeed)
				gen.dwell(members[1], day, 18, 42, 18, 56, mid) // 14-minute unknown stop
				gen.trip(members[1], day, 18, 57, mid, home, driveSpeed)
			} else {
				gen.trip(members[1], day, 18, 18, office, home, driveSpeed)
			}
		} else {
			gen.dwell(members[1], day, 9, 0, 10, 30, home)
			gen.trip(members[1], day, 10, 32, home, market, driveSpeed)
			gen.dwell(members[1], day, 10, 45, 11, 25, market)
			gen.trip(members[1], day, 11, 27, market, home, driveSpeed)
		}

		// Mom: home base, market late morning, park on weekends.
		gen.dwell(members[2], day, 8, 0, 10, 55, home)
		gen.trip(members[2], day, 11, 0, home, market, walkSpeed)
		gen.dwell(members[2], day, 11, 20, 12, 5, market)
		gen.trip(members[2], day, 12, 8, market, home, walkSpeed)
		gen.dwell(members[2], day, 12, 30, 21, 0, home)
	}

	// Everyone gets a fresh "now" position so current-position queries feel live.
	gen.point(members[1], now.Add(-90*time.Second), office.lat, office.lng, 0) // Dad at office
	gen.point(members[2], now.Add(-2*time.Minute), home.lat, home.lng, 0)     // Mom home
	gen.point(members[3], now.Add(-3*time.Minute), school.lat, school.lng, 0) // Zara at school
	gen.point(me, now.Add(-60*time.Second), home.lat, home.lng, 0)            // demo user home

	gen.flush()
	gen.writeDailyActivity()

	// ── Geofence events derived from Zara's & Dad's schedule ──
	fmt.Println("writing geofence events...")
	for dayOffset := -6; dayOffset <= 0; dayOffset++ {
		day := now.AddDate(0, 0, dayOffset)
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}
		events := []struct {
			uid   string
			fence place
			typ   string
			h, m  int
		}{
			{members[3].id, home, "exit", 7, 55},
			{members[3].id, school, "enter", 8, 23},
			{members[3].id, school, "exit", 15, 32},
			{members[3].id, home, "enter", 15, 58},
			{members[1].id, home, "exit", 9, 5},
			{members[1].id, home, "enter", 18, 44},
		}
		for _, e := range events {
			at := time.Date(day.Year(), day.Month(), day.Day(), e.h, e.m, 0, 0, day.Location())
			if at.After(now) {
				continue
			}
			if _, err := pool.DB.ExecContext(ctx, `
				INSERT INTO geofence_events (user_id, fence_name, event_type, lat, lng, ts)
				VALUES ($1, $2, $3, $4, $5, $6)`,
				e.uid, e.fence.name, e.typ, e.fence.lat, e.fence.lng, at.UnixMilli()); err != nil {
				fail("geofence event: %v", err)
			}
		}
	}

	fmt.Printf(`
done: %d position points across 4 users, 7 days.

Demo login:  demo@kinnect.app / %s
Family:      Dad, Mom, Zara (contacts of the demo user)
Try asking:  "Where is everyone right now?"
             "Where did Dad stop on his way home on Wednesday?"
             "When did Zara get to school today?"
             "How long was Zara at the park this week?"
`, gen.total, demoPass)
}

// ─── Trace generation ────────────────────────────────────────────────────────

type generator struct {
	ctx   context.Context
	db    *sql.DB
	rng   *rand.Rand
	batch []posRow
	total int
	// last point per user for daily aggregation
	last map[string]posRow
	// day aggregates: uid -> date -> {distance m, active seconds}
	days map[string]map[string]*dayAgg
}

type dayAgg struct {
	distM   float64
	activeS float64
}

type posRow struct {
	uid      string
	lat, lng float64
	speed    float64
	ts       int64
}

// trip emits points along home→dest at speed, sampled every ~40s with jitter.
// Points after time.Now() are skipped so "today" truncates naturally.
func (g *generator) trip(m *member, day time.Time, h, min int, from, to place, speed float64) {
	start := time.Date(day.Year(), day.Month(), day.Day(), h, min, 0, 0, day.Location())
	dist := haversineM(from.lat, from.lng, to.lat, to.lng)
	durS := dist / speed
	steps := int(durS/40) + 2
	for i := 0; i <= steps; i++ {
		f := float64(i) / float64(steps)
		at := start.Add(time.Duration(f*durS) * time.Second)
		if at.After(time.Now()) {
			return
		}
		// Slight arc + jitter so routes don't look laser-straight.
		arc := 0.0015 * math.Sin(f*math.Pi)
		lat := from.lat + (to.lat-from.lat)*f + arc + g.jitter(0.00012)
		lng := from.lng + (to.lng-from.lng)*f + arc/2 + g.jitter(0.00012)
		g.point(m, at, lat, lng, speed*(0.75+g.rng.Float64()*0.5))
	}
}

// dwell emits sparse stationary points inside the place radius.
func (g *generator) dwell(m *member, day time.Time, h1, m1, h2, m2 int, p place) {
	start := time.Date(day.Year(), day.Month(), day.Day(), h1, m1, 0, 0, day.Location())
	end := time.Date(day.Year(), day.Month(), day.Day(), h2, m2, 0, 0, day.Location())
	for at := start; at.Before(end); at = at.Add(time.Duration(3+g.rng.Intn(3)) * time.Minute) {
		if at.After(time.Now()) {
			return
		}
		g.point(m, at, p.lat+g.jitter(0.00018), p.lng+g.jitter(0.00018), 0)
	}
}

func (g *generator) point(m *member, at time.Time, lat, lng, speed float64) {
	row := posRow{m.id, lat, lng, speed, at.UnixMilli()}
	g.batch = append(g.batch, row)

	// Daily aggregation for daily_activity.
	if g.last == nil {
		g.last = make(map[string]posRow)
		g.days = make(map[string]map[string]*dayAgg)
	}
	date := at.Format("2006-01-02")
	if prev, ok := g.last[m.id]; ok && time.UnixMilli(prev.ts).Format("2006-01-02") == date {
		if d := haversineM(prev.lat, prev.lng, lat, lng); d > 5 {
			if g.days[m.id] == nil {
				g.days[m.id] = make(map[string]*dayAgg)
			}
			if g.days[m.id][date] == nil {
				g.days[m.id][date] = &dayAgg{}
			}
			agg := g.days[m.id][date]
			agg.distM += d
			if dt := float64(row.ts-prev.ts) / 1000; dt > 0 && dt < 180 && speed > 0.5 {
				agg.activeS += dt
			}
		}
	}
	g.last[m.id] = row

	if len(g.batch) >= 500 {
		g.flush()
	}
}

// writeDailyActivity persists the aggregated per-day summaries.
func (g *generator) writeDailyActivity() {
	n := 0
	for uid, byDate := range g.days {
		for date, agg := range byDate {
			if _, err := g.db.ExecContext(g.ctx, `
				INSERT INTO daily_activity (user_id, date, distance_m, active_minutes, updated_at)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (user_id, date) DO UPDATE
					SET distance_m = EXCLUDED.distance_m, active_minutes = EXCLUDED.active_minutes`,
				uid, date, int(agg.distM), int(agg.activeS/60), time.Now().UnixMilli()); err != nil {
				fail("daily activity: %v", err)
			}
			n++
		}
	}
	fmt.Printf("daily_activity: %d rows\n", n)
}

func (g *generator) flush() {
	if len(g.batch) == 0 {
		return
	}
	tx, err := g.db.BeginTx(g.ctx, nil)
	if err != nil {
		fail("tx: %v", err)
	}
	stmt, err := tx.PrepareContext(g.ctx,
		`INSERT INTO position_history (user_id, lat, lng, speed, ts) VALUES ($1, $2, $3, $4, $5)`)
	if err != nil {
		fail("prepare: %v", err)
	}
	for _, r := range g.batch {
		if _, err := stmt.ExecContext(g.ctx, r.uid, r.lat, r.lng, r.speed, r.ts); err != nil {
			fail("insert point: %v", err)
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		fail("commit: %v", err)
	}
	g.total += len(g.batch)
	g.batch = g.batch[:0]
}

func (g *generator) jitter(scale float64) float64 {
	return (g.rng.Float64()*2 - 1) * scale
}

func haversineM(lat1, lng1, lat2, lng2 float64) float64 {
	const r = 6371000.0
	p1, p2 := lat1*math.Pi/180, lat2*math.Pi/180
	dp, dl := (lat2-lat1)*math.Pi/180, (lng2-lng1)*math.Pi/180
	a := math.Sin(dp/2)*math.Sin(dp/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return 2 * r * math.Asin(math.Min(1, math.Sqrt(a)))
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "seed-demo: "+format+"\n", args...)
	os.Exit(1)
}
