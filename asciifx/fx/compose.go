package fx

// Sequence plays finite effects one after another, each seeing its own tick
// count from zero. Transitions each receive the same target content.
func Sequence(effects ...Finite) Finite { return &sequence{effects: effects} }

type sequence struct{ effects []Finite }

func (s *sequence) Duration() float64 {
	var d float64
	for _, e := range s.effects {
		d += e.Duration()
	}
	return d
}

func (s *sequence) Step(f *Frame) {
	start := 0.0
	for i, e := range s.effects {
		end := start + e.Duration()
		if f.T() < end || i == len(s.effects)-1 {
			local := *f
			local.Tick = f.Tick - int(start/f.Dt+0.5)
			e.Step(&local)
			return
		}
		start = end
	}
}

// Parallel steps every effect on the same buffer in order, so later effects
// see earlier effects' output. Its duration is the longest child's.
func Parallel(effects ...Effect) Effect { return &parallel{effects: effects} }

type parallel struct{ effects []Effect }

func (p *parallel) Step(f *Frame) {
	for _, e := range p.effects {
		e.Step(f)
	}
}

func (p *parallel) Duration() float64 {
	var d float64
	for _, e := range p.effects {
		fin, ok := e.(Finite)
		if !ok {
			return 0
		}
		d = max(d, fin.Duration())
	}
	return d
}

// Delay holds the target content untouched for seconds, then runs e.
func Delay(seconds float64, e Finite) Finite { return &delay{d: seconds, e: e} }

type delay struct {
	d float64
	e Finite
}

func (d *delay) Duration() float64 { return d.d + d.e.Duration() }

func (d *delay) Step(f *Frame) {
	if f.T() < d.d {
		return
	}
	local := *f
	local.Tick = f.Tick - int(d.d/f.Dt+0.5)
	d.e.Step(&local)
}

// Hold keeps e's final frame on screen for an extra number of seconds.
func Hold(e Finite, seconds float64) Finite { return &hold{e: e, d: seconds} }

type hold struct {
	e Finite
	d float64
}

func (h *hold) Duration() float64 { return h.e.Duration() + h.d }

func (h *hold) Step(f *Frame) {
	local := *f
	if last := int(h.e.Duration()/f.Dt + 0.5); local.Tick > last {
		local.Tick = last
	}
	h.e.Step(&local)
}
