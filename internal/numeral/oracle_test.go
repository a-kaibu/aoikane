package numeral

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type oracleCand struct{ Reading, Pronunciation, Accent, Connection string }
type oracleMorph struct {
	S     string        `json:"s"`
	POS   [3]uint32     `json:"pos"`
	Cands []oracleCand  `json:"c"`
	Chain []oracleMorph `json:"chain"`
}
type oracleRow struct {
	Text string        `json:"text"`
	In   []oracleMorph `json:"in"`
	Out  []oracleMorph `json:"out"`
}

// oracleCase is one numeral run the engine merged: the input morphs and the
// expected groups of every candidate.
type oracleCase struct {
	Text    string
	In      []oracleMorph
	Out     oracleMorph
	Number  string
	Counter string
}

func loadOracle(t *testing.T) []oracleCase {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "private", "num_oracle.jsonl"))
	if err != nil {
		t.Skip("numeral oracle capture is not available")
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<24), 1<<24)
	var cases []oracleCase
	for sc.Scan() {
		var r oracleRow
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		i := 0
		for i < len(r.In) && i < len(r.Out) && eq(r.In[i], r.Out[i]) {
			i++
		}
		j := 0
		for j < len(r.In)-i && j < len(r.Out)-i && eq(r.In[len(r.In)-1-j], r.Out[len(r.Out)-1-j]) {
			j++
		}
		in, out := r.In[i:len(r.In)-j], r.Out[i:len(r.Out)-j]
		if len(in) == 0 || len(out) != 1 {
			continue
		}
		c := oracleCase{Text: r.Text, In: in, Out: out[0]}
		var sb strings.Builder
		for _, m := range in {
			sb.WriteString(m.S)
		}
		if sb.String() != out[0].S {
			continue
		}
		c.Number = sb.String()
		cases = append(cases, c)
	}
	return cases
}

func eq(a, b oracleMorph) bool {
	if a.S != b.S || len(a.Cands) != len(b.Cands) || a.POS != b.POS {
		return false
	}
	for i := range a.Cands {
		if a.Cands[i] != b.Cands[i] {
			return false
		}
	}
	return len(b.Chain) == 0
}

// groupsOf lists the accent groups of a candidate across the morph and its chain.
func groupsOf(m oracleMorph, cand int) ([]string, string) {
	var out []string
	read := ""
	for _, x := range append([]oracleMorph{m}, m.Chain...) {
		if cand >= len(x.Cands) {
			continue
		}
		c := x.Cands[cand]
		read += c.Reading
		out = append(out, strings.Split(c.Accent, ",")...)
	}
	return out, read
}

func formatGroups(gs []Group) []string {
	var out []string
	for _, g := range gs {
		if g.Pause {
			out = append(out, "s0")
			continue
		}
		out = append(out, fmt.Sprintf("%d-%d-%d", g.Up, g.Down, g.Moras))
	}
	return out
}

// TestPlainNumbers compares the integer readings (no counter) with the engine.
func TestPlainNumbers(t *testing.T) {
	r := openRules(t, "bancharcx-standard.txt")
	total, bad := 0, 0
	for _, c := range loadOracle(t) {
		if _, err := Value(c.Number); err != nil {
			continue
		}
		wantAcc, wantRead := groupsOf(c.Out, 0)
		got, err := r.Read(Request{Number: c.Number})
		if err != nil {
			continue
		}
		total++
		var read string
		for _, g := range got[0].Groups {
			read += g.Read
		}
		gotAcc := formatGroups(got[0].Groups)
		if read != wantRead || strings.Join(gotAcc, ",") != strings.Join(wantAcc, ",") {
			bad++
			t.Logf("%s: got %s %v want %s %v", c.Number, read, gotAcc, wantRead, wantAcc)
		}
	}
	t.Logf("%d plain numbers, %d differ", total, bad)
}

// TestCounterNumbers compares numbers followed by a counter with the engine.
func TestCounterNumbers(t *testing.T) {
	r := openRules(t, "bancharcx-standard.txt")
	total, bad := 0, 0
	byKind := map[string]int{}
	for _, c := range loadOracle(t) {
		last := c.In[len(c.In)-1]
		if len(c.In) < 2 {
			continue
		}
		if _, ok := r.Counters[last.S]; !ok {
			continue
		}
		num := strings.TrimSuffix(c.Number, last.S)
		if _, err := Value(num); err != nil {
			continue
		}
		var fall, up, moras int
		if len(last.Cands) > 0 {
			fmt.Sscanf(last.Cands[0].Accent, "%d-%d-%d", &up, &fall, &moras)
		}
		got, err := r.Read(Request{Number: num, Counter: last.S, CounterFall: fall, CounterMoras: moras, CounterHasInfo: true})
		if err != nil {
			t.Logf("%s: %v", c.Number, err)
			continue
		}
		total++
		ok := len(got) == len(c.Out.Cands)+len(c.Out.Chain)*0
		for k := range got {
			wantAcc, wantRead := groupsOf(c.Out, k)
			var read string
			for _, g := range got[k].Groups {
				read += g.Read
			}
			if k >= len(c.Out.Cands) || read != wantRead || strings.Join(formatGroups(got[k].Groups), ",") != strings.Join(wantAcc, ",") {
				ok = false
				if k == 0 {
					byKind[last.S]++
					if bad < 60 {
						t.Logf("%s: got %s %v want %s %v", c.Number, read, formatGroups(got[k].Groups), wantRead, wantAcc)
					}
				}
			}
		}
		if !ok {
			bad++
		}
	}
	t.Logf("%d counter numbers, %d differ", total, bad)
}

// TestDecimals compares decimal numbers (with and without a counter).
func TestDecimals(t *testing.T) {
	r := openRules(t, "bancharcx-standard.txt")
	total, bad := 0, 0
	for _, c := range loadOracle(t) {
		if !strings.Contains(c.Number, "．") {
			continue
		}
		last := c.In[len(c.In)-1]
		num, counter := c.Number, ""
		req := Request{}
		if _, ok := r.Counters[last.S]; ok {
			num, counter = strings.TrimSuffix(c.Number, last.S), last.S
			var up, fall, moras int
			if len(last.Cands) > 0 {
				fmt.Sscanf(last.Cands[0].Accent, "%d-%d-%d", &up, &fall, &moras)
			}
			req = Request{Counter: counter, CounterFall: fall, CounterMoras: moras, CounterHasInfo: true}
		}
		req.Number = num
		got, err := r.Read(req)
		if err != nil {
			t.Logf("%s: %v", c.Number, err)
			continue
		}
		total++
		wantAcc, wantRead := groupsOf(c.Out, 0)
		var read string
		for _, g := range got[0].Groups {
			read += g.Read
		}
		if read != wantRead || strings.Join(formatGroups(got[0].Groups), ",") != strings.Join(wantAcc, ",") {
			bad++
			t.Logf("%s: got %s %v want %s %v", c.Number, read, formatGroups(got[0].Groups), wantRead, wantAcc)
		}
	}
	t.Logf("%d decimals, %d differ", total, bad)
}

func TestZZRead(t *testing.T) {
	r := openRules(t, "bancharcx-standard.txt")
	for _, n := range []string{"５００万", "３万", "５００"} {
		got, err := r.Read(Request{Number: n, Counter: "円", CounterFall: 1, CounterMoras: 2})
		fmt.Println(n, got, err)
	}
}

// TestPatterns compares ranges, fractions and times with the engine.
func TestPatterns(t *testing.T) {
	r := openRules(t, "bancharcx-standard.txt")
	total, bad := 0, 0
	for _, c := range loadOracle(t) {
		n := c.Number
		var got []Reading
		var err error
		switch {
		case strings.Contains(n, "～"):
			p := strings.SplitN(n, "～", 2)
			req := Request{Number: p[1]}
			last := c.In[len(c.In)-1]
			if _, ok := r.Counters[last.S]; ok {
				req.Number, req.Counter = strings.TrimSuffix(p[1], last.S), last.S
				var up, fall, moras int
				fmt.Sscanf(last.Cands[0].Accent, "%d-%d-%d", &up, &fall, &moras)
				req.CounterFall, req.CounterMoras = fall, moras
			}
			got, err = r.ReadRange(p[0], "カラ", req)
		case strings.Contains(n, "対"):
			p := strings.SplitN(n, "対", 2)
			got, err = r.ReadRange(p[0], "タイ", Request{Number: p[1]})
		case strings.Contains(n, "／") && strings.Count(n, "／") == 1:
			p := strings.SplitN(n, "／", 2)
			got, err = r.ReadFraction(p[0], p[1])
		case strings.Contains(n, "："):
			got, err = r.ReadTime(strings.Split(n, "：")...)
		default:
			continue
		}
		if err != nil {
			t.Logf("%s: %v", n, err)
			continue
		}
		total++
		wantAcc, wantRead := groupsOf(c.Out, 0)
		var read string
		for _, g := range got[0].Groups {
			if !g.Silent {
				read += g.Read
			}
		}
		wantRead = strings.ReplaceAll(wantRead, "ｂ", "")
		read = strings.ReplaceAll(read, "ｂ", "")
		if read != wantRead || strings.Join(formatGroups(got[0].Groups), ",") != strings.Join(wantAcc, ",") {
			bad++
			t.Logf("%s: got %s %v want %s %v", n, read, formatGroups(got[0].Groups), wantRead, wantAcc)
		}
	}
	t.Logf("%d patterns, %d differ", total, bad)
}
