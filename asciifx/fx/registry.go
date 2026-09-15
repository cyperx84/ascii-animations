package fx

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/cyperx84/ascii-animations/asciifx/ease"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

// Kind groups effects by how they use the buffer.
type Kind string

const (
	// Ambient effects fill the whole buffer and usually loop forever.
	Ambient Kind = "ambient"
	// Transition effects animate target content in or out.
	Transition Kind = "transition"
	// Spinner effects are small looping indicators, typically one line.
	Spinner Kind = "spinner"
)

// ParamType names the accepted value type of a Param.
type ParamType string

const (
	Int          ParamType = "int"
	Float        ParamType = "float"
	Bool         ParamType = "bool"
	String       ParamType = "string"
	Enum         ParamType = "enum"
	Palette      ParamType = "palette"
	Easing       ParamType = "easing"
	PatternParam ParamType = "pattern"
	ColorParam   ParamType = "color"
)

// Param describes one tunable input of an effect.
type Param struct {
	Name    string    `json:"name"`
	Type    ParamType `json:"type"`
	Default string    `json:"default"`
	Doc     string    `json:"doc"`
	Min     *float64  `json:"min,omitempty"`
	Max     *float64  `json:"max,omitempty"`
	Options []string  `json:"options,omitempty"`
}

// FloatParam declares a float in [lo, hi].
func FloatParam(name string, def, lo, hi float64, doc string) Param {
	return Param{Name: name, Type: Float, Default: strconv.FormatFloat(def, 'g', -1, 64), Doc: doc, Min: &lo, Max: &hi}
}

// IntParam declares an int in [lo, hi].
func IntParam(name string, def, lo, hi int, doc string) Param {
	l, h := float64(lo), float64(hi)
	return Param{Name: name, Type: Int, Default: strconv.Itoa(def), Doc: doc, Min: &l, Max: &h}
}

// BoolParam declares a boolean.
func BoolParam(name string, def bool, doc string) Param {
	return Param{Name: name, Type: Bool, Default: strconv.FormatBool(def), Doc: doc}
}

// EnumParam declares a choice between options.
func EnumParam(name, def string, options []string, doc string) Param {
	return Param{Name: name, Type: Enum, Default: def, Doc: doc, Options: options}
}

// StringParam declares free text.
func StringParam(name, def, doc string) Param {
	return Param{Name: name, Type: String, Default: def, Doc: doc}
}

// PaletteParam declares a palette name or hex stop list.
func PaletteParam(name, def, doc string) Param {
	return Param{Name: name, Type: Palette, Default: def, Doc: doc, Options: tint.PaletteNames()}
}

// EasingParam declares an easing curve.
func EasingParam(name, def, doc string) Param {
	return Param{Name: name, Type: Easing, Default: def, Doc: doc, Options: ease.Names()}
}

// PatternParamOf declares a spatial pattern.
func PatternParamOf(name, def, doc string) Param {
	return Param{Name: name, Type: PatternParam, Default: def, Doc: doc, Options: PatternNames()}
}

// ColorParamOf declares a single hex colour; "none" means terminal default.
func ColorParamOf(name, def, doc string) Param {
	return Param{Name: name, Type: ColorParam, Default: def, Doc: doc}
}

// Spec registers an effect and everything a tool or agent needs to use it.
type Spec struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Kind        Kind     `json:"kind"`
	Tags        []string `json:"tags,omitempty"`
	// Glyphs names the character classes the effect emits: ascii, box,
	// block, halfblock, braille. Agents use it to judge terminal safety.
	Glyphs []string `json:"glyphs"`
	// FPS is the recommended tick rate.
	FPS int `json:"fps"`
	// Duration in seconds for finite effects; 0 means it loops forever.
	Duration float64 `json:"duration"`
	// Size is the smallest sensible and the suggested default size.
	MinW   int     `json:"min_w"`
	MinH   int     `json:"min_h"`
	DefW   int     `json:"default_w"`
	DefH   int     `json:"default_h"`
	Params []Param `json:"params"`
	// Content says whether the effect animates target content.
	Content bool `json:"content"`
	// Example is a ready-to-run CLI invocation.
	Example string `json:"example"`

	New func(p Values, w, h int, rng *rand.Rand) (Effect, error) `json:"-"`
}

var (
	regMu    sync.RWMutex
	registry = map[string]*Spec{}
)

// Register adds a spec. It panics on duplicate names or invalid defaults, so
// mistakes surface at init time rather than in a user's terminal.
func Register(s Spec) {
	regMu.Lock()
	defer regMu.Unlock()
	if _, dup := registry[s.Name]; dup {
		panic("asciifx: duplicate effect " + s.Name)
	}
	if s.New == nil {
		panic("asciifx: effect " + s.Name + " has no constructor")
	}
	if s.FPS == 0 {
		s.FPS = 30
	}
	if s.DefW == 0 {
		s.DefW, s.DefH = 60, 16
	}
	sp := &s
	if _, err := sp.Resolve(nil); err != nil {
		panic("asciifx: effect " + s.Name + " has invalid defaults: " + err.Error())
	}
	registry[s.Name] = sp
}

// Lookup returns the spec for name.
func Lookup(name string) (*Spec, error) {
	regMu.RLock()
	defer regMu.RUnlock()
	if s, ok := registry[name]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("unknown effect %q: run `asciifx list` to see available effects", name)
}

// All returns every registered spec sorted by kind then name.
func All() []*Spec {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]*Spec, 0, len(registry))
	for _, s := range registry {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// Values is a validated, fully defaulted parameter set.
type Values struct {
	spec *Spec
	raw  map[string]string
}

// Resolve validates raw key=value params against the spec and fills defaults.
// Unknown keys are errors, so a typo never silently does nothing.
func (s *Spec) Resolve(raw map[string]string) (Values, error) {
	v := Values{spec: s, raw: map[string]string{}}
	for k, val := range raw {
		p := s.param(k)
		if p == nil {
			names := make([]string, len(s.Params))
			for i, p := range s.Params {
				names[i] = p.Name
			}
			return v, fmt.Errorf("effect %s has no param %q (params: %s)", s.Name, k, strings.Join(names, ", "))
		}
		if err := p.validate(val); err != nil {
			return v, fmt.Errorf("param %s: %w", k, err)
		}
		v.raw[k] = val
	}
	for _, p := range s.Params {
		if _, ok := v.raw[p.Name]; !ok {
			if err := p.validate(p.Default); err != nil {
				return v, fmt.Errorf("default for %s: %w", p.Name, err)
			}
			v.raw[p.Name] = p.Default
		}
	}
	return v, nil
}

func (s *Spec) param(name string) *Param {
	for i := range s.Params {
		if s.Params[i].Name == name {
			return &s.Params[i]
		}
	}
	return nil
}

func (p *Param) validate(val string) error {
	switch p.Type {
	case Int, Float:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) || (p.Type == Int && f != math.Trunc(f)) {
			return fmt.Errorf("want a finite %s, got %q", p.Type, val)
		}
		if p.Min != nil && f < *p.Min {
			return fmt.Errorf("%g is below the minimum %g", f, *p.Min)
		}
		if p.Max != nil && f > *p.Max {
			return fmt.Errorf("%g is above the maximum %g", f, *p.Max)
		}
	case Bool:
		if _, err := strconv.ParseBool(val); err != nil {
			return fmt.Errorf("want true or false, got %q", val)
		}
	case Enum:
		for _, o := range p.Options {
			if o == val {
				return nil
			}
		}
		return fmt.Errorf("want one of %s, got %q", strings.Join(p.Options, ", "), val)
	case Palette:
		_, err := tint.ParsePalette(val)
		return err
	case Easing:
		_, err := ease.Parse(val)
		return err
	case PatternParam:
		_, err := ParsePattern(val, 0)
		return err
	case ColorParam:
		if val == "" || val == "none" {
			return nil
		}
		_, err := tint.Hex(val)
		return err
	}
	return nil
}

// String returns the raw value of a param.
func (v Values) String(name string) string { return v.raw[name] }

// Int returns an int param. It panics on names the spec does not declare,
// which is a programming error in the effect.
func (v Values) Int(name string) int { return int(v.Float(name)) }

// Float returns a numeric param.
func (v Values) Float(name string) float64 {
	f, err := strconv.ParseFloat(v.must(name), 64)
	if err != nil {
		panic(err)
	}
	return f
}

// Bool returns a boolean param.
func (v Values) Bool(name string) bool {
	b, _ := strconv.ParseBool(v.must(name))
	return b
}

// Palette returns a palette param.
func (v Values) Palette(name string) tint.Gradient {
	g, _ := tint.ParsePalette(v.must(name))
	return g
}

// Easing returns an easing param.
func (v Values) Easing(name string) ease.Func {
	f, _ := ease.Parse(v.must(name))
	return f
}

// Pattern returns a pattern param seeded with seed.
func (v Values) Pattern(name string, seed uint64) Pattern {
	p, _ := ParsePattern(v.must(name), seed)
	return p
}

// Color returns a colour param; "none" or empty is tint.None.
func (v Values) Color(name string) tint.Color {
	c, _ := tint.Hex(v.must(name))
	return c
}

// Map returns the resolved params as a plain map.
func (v Values) Map() map[string]string {
	out := make(map[string]string, len(v.raw))
	for k, val := range v.raw {
		out[k] = val
	}
	return out
}

func (v Values) must(name string) string {
	val, ok := v.raw[name]
	if !ok {
		panic(fmt.Sprintf("asciifx: effect %s reads undeclared param %q", v.spec.Name, name))
	}
	return val
}
