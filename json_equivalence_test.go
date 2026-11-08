package orderedmap

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"math"
	"reflect"
	"testing"
)

type equivalenceInt int16
type equivalenceString string

// marshalCases are maps whose bytes must equal the upstream v2.1.8 bytes.
func marshalCases() map[string]any {
	strings := New[string, any]()
	strings.Set("plain", "v")
	strings.Set("html <&> \u2028\u2029", "<b>&amp;</b> \u2028")
	strings.Set("ctl \x00\x01\x1f\t\n\r\"\\/", "ctl \x00\x7f\t\n\r\"\\/")
	strings.Set("utf8 èé😀", "invalid \xff\xfe end")
	strings.Set("nil slice", []int(nil))
	strings.Set("nil map", map[string]int(nil))
	strings.Set("null", nil)
	strings.Set("numbers", []any{0, -1, 1.5, math.MaxInt64, uint64(math.MaxUint64), 1e21, 1e-7, float32(0.1), json.Number("12345678901234567890")})
	strings.Set("raw", json.RawMessage(` {"x" : [1, 2]} `))
	strings.Set("bytes", []byte("abc"))
	strings.Set("struct", struct {
		A int    `json:"a,omitempty"`
		B string `json:"b"`
	}{B: "<x>"})
	strings.Set("plain", "updated keeps position")
	nested := New[string, any]()
	nested.Set("z", 1)
	nested.Set("a", New[int, string]())
	strings.Set("nested", nested)
	strings.Set("nil ordered map", (*OrderedMap[string, int])(nil))

	ints := New[int, any]()
	ints.Set(-7, "neg")
	ints.Set(math.MaxInt, "max")
	ints.Set(0, "zero")

	uints := New[uint64, string]()
	uints.Set(math.MaxUint64, "max")
	uints.Set(3, "three")

	small := New[equivalenceInt, bool]()
	small.Set(-32768, true)
	small.Set(12, false)

	named := New[equivalenceString, float64]()
	named.Set("b", 2.25)
	named.Set("a", -0.0)

	text := New[marshallable, int]()
	text.Set(4, 40)
	text.Set(-2, 20)

	return map[string]any{
		"strings": strings, "ints": ints, "uints": uints, "small": small,
		"named": named, "text": text, "empty": New[string, int](), "zero": &OrderedMap[string, int]{},
	}
}

// referenceOf wraps a map in the upstream v2.1.8 codec.
func referenceOf(om any) any {
	switch m := om.(type) {
	case *OrderedMap[string, any]:
		return referenceMap[string, any]{m}
	case *OrderedMap[int, any]:
		return referenceMap[int, any]{m}
	case *OrderedMap[uint64, string]:
		return referenceMap[uint64, string]{m}
	case *OrderedMap[equivalenceInt, bool]:
		return referenceMap[equivalenceInt, bool]{m}
	case *OrderedMap[equivalenceString, float64]:
		return referenceMap[equivalenceString, float64]{m}
	case *OrderedMap[marshallable, int]:
		return referenceMap[marshallable, int]{m}
	case *OrderedMap[string, int]:
		return referenceMap[string, int]{m}
	}
	panic("no reference for " + reflect.TypeOf(om).String())
}

// TestMarshalMatchesUpstream writes each map with the json/v2 codec and
// with the upstream jsonparser/easyjson codec: MarshalJSON called
// directly, encoding/json (alone and inside a struct) and json/v2 with the
// encoding/json options give the same bytes or both fail.
func TestMarshalMatchesUpstream(t *testing.T) {
	type holder struct {
		M any `json:"m"`
		N int `json:"n"`
	}
	for name, om := range marshalCases() {
		ref := referenceOf(om)
		direct := func(v any) ([]byte, error) { return v.(json.Marshaler).MarshalJSON() }
		ways := map[string]func(any) ([]byte, error){
			"MarshalJSON": direct,
			"encoding/json": func(v any) ([]byte, error) {
				return json.Marshal(v)
			},
			"encoding/json in a struct": func(v any) ([]byte, error) {
				return json.Marshal(holder{M: v, N: 1})
			},
			"json/v2 with v1 options": func(v any) ([]byte, error) {
				return jsonv2.Marshal(v, json.DefaultOptionsV1())
			},
		}
		for way, marshal := range ways {
			got, gotErr := marshal(om)
			want, wantErr := marshal(ref)
			if (gotErr == nil) != (wantErr == nil) || string(got) != string(want) {
				t.Errorf("%s through %s:\n got %s (%v)\nwant %s (%v)", name, way, got, gotErr, want, wantErr)
			}
		}
	}
}

type pairOf[K comparable, V any] struct {
	Key   K
	Value V
}

func pairsOf[K comparable, V any](om *OrderedMap[K, V]) []pairOf[K, V] {
	if om == nil || om.list == nil {
		return nil
	}
	var out []pairOf[K, V]
	for p := om.Oldest(); p != nil; p = p.Next() {
		out = append(out, pairOf[K, V]{p.Key, p.Value})
	}
	return out
}

var unmarshalInputs = []string{
	`{"b":1,"a":[1,2.5,"x",null,true,{"k":"v"}],"c":{"z":0,"y":-1e3}}`,
	`{"a":1,"b":2,"a":3}`,
	`{"k":12345678901234567890,"f":1.7976931348623157e308,"s":5e-324,"n":-0}`,
	`{"html":"<&>","esc":"\u00e8\ud83d\ude00\n\t\"\\\/\u0000","nul\u0000key":1}`,
	`{"esc\u0041ped\n":"key","\u00e8":2}`,
	"{\"bad\xffkey\":1}",
	"{\"k\":\"bad\xffvalue\"}",
	`{"lone":"\ud800"}`,
	`{"\ud800":"lone key"}`,
	`{}`,
	` { "a" : 1 } `,
	`null`,
	`[]`,
	`[1,2]`,
	`"string"`,
	`12`,
	`true`,
	`{"a":}`,
	`{"a":1,}`,
	`{"a" 1}`,
	`{"a":1`,
	`{"a":1}}`,
	`{"a":1} {"b":2}`,
	`{'a':1}`,
	`{"a":tru}`,
	`{"a":"\x"}`,
	`{"1":"x","-2":"y","007":"z"}`,
	`{"1.5":"x"}`,
	`{"99999999999999999999":"x"}`,
	`{"40000":"x"}`,
	`{"#1#":1,"#-3#":2}`,
	`{"nope":1}`,
}

// unmarshalBoth decodes in into a fresh map with the json/v2 codec and with
// the upstream codec, directly (UnmarshalJSON) or through encoding/json.
func unmarshalBoth[K comparable, V any](t *testing.T, in string, viaEncodingJSON bool) {
	t.Helper()
	got, want := New[K, V](), New[K, V]()
	var gotErr, wantErr error
	if viaEncodingJSON {
		gotErr = json.Unmarshal([]byte(in), got)
		wantErr = json.Unmarshal([]byte(in), &referenceMap[K, V]{want})
	} else {
		gotErr = got.UnmarshalJSON([]byte(in))
		wantErr = (&referenceMap[K, V]{want}).UnmarshalJSON([]byte(in))
	}
	if (gotErr == nil) != (wantErr == nil) {
		t.Errorf("%q into %T (encoding/json %v): error %v, upstream %v", in, got, viaEncodingJSON, gotErr, wantErr)
		return
	}
	if gotErr == nil && !reflect.DeepEqual(pairsOf(got), pairsOf(want)) {
		t.Errorf("%q into %T (encoding/json %v):\n got %#v\nwant %#v", in, got, viaEncodingJSON, pairsOf(got), pairsOf(want))
	}
}

// TestUnmarshalMatchesUpstream decodes each input with the json/v2 codec and
// with the upstream codec into maps of several key and value types: equal
// pairs in equal order, or an error in both. Divergences of the upstream
// codec on input it accepted although malformed are listed in
// knownUnmarshalDivergences and checked to stay limited to those inputs.
func TestUnmarshalMatchesUpstream(t *testing.T) {
	for _, in := range unmarshalInputs {
		for _, via := range []bool{false, true} {
			if knownUnmarshalDivergences[in] && !via {
				continue
			}
			unmarshalBoth[string, any](t, in, via)
			unmarshalBoth[string, json.RawMessage](t, in, via)
			unmarshalBoth[equivalenceString, any](t, in, via)
			unmarshalBoth[int, string](t, in, via)
			unmarshalBoth[equivalenceInt, string](t, in, via)
			unmarshalBoth[uint8, any](t, in, via)
			unmarshalBoth[marshallable, int](t, in, via)
		}
	}
}

// knownUnmarshalDivergences are inputs that the upstream UnmarshalJSON,
// called directly with the bytes, accepted although they are not one JSON
// object (jsonparser stops at the first object and skips what it does not
// parse); the json/v2 codec reports an error. Through encoding/json, which
// validates the input before calling the method, both report an error.
var knownUnmarshalDivergences = map[string]bool{
	`{"a":1,}`:        true,
	`{"a":1}}`:        true,
	`{"a":1} {"b":2}`: true,
}

// TestInvalidUTF8KeyReadsTheSame: a string key with invalid UTF-8 is the
// one written differently: upstream (easyjson) wrote each invalid byte as
// the escape \ufffd, json/v2 writes U+FFFD itself, as encoding/json does
// for values and map keys. Both bytes read back as the same pairs.
func TestInvalidUTF8KeyReadsTheSame(t *testing.T) {
	om := New[string, any]()
	om.Set("invalid key \xff\xfe", 1)
	om.Set("ok", "\xff")
	got, err := json.Marshal(om)
	if err != nil {
		t.Fatal(err)
	}
	want, err := json.Marshal(referenceMap[string, any]{om})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "{\"invalid key \ufffd\ufffd\":1,\"ok\":\"\ufffd\"}" || string(want) != `{"invalid key \ufffd\ufffd":1,"ok":"`+"\ufffd"+`"}` {
		t.Fatalf("got %s, upstream %s", got, want)
	}
	a, b := New[string, any](), New[string, any]()
	if err := json.Unmarshal(got, a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, b); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pairsOf(a), pairsOf(b)) {
		t.Fatalf("read back %v, upstream %v", pairsOf(a), pairsOf(b))
	}
}

// TestUnmarshalDivergencesAreOnlyMalformedInput proves the divergences
// listed above: upstream accepts them directly, json/v2 refuses them, and
// encoding/json refuses them before either codec runs.
func TestUnmarshalDivergencesAreOnlyMalformedInput(t *testing.T) {
	for in := range knownUnmarshalDivergences {
		ref := New[string, any]()
		if err := (&referenceMap[string, any]{ref}).UnmarshalJSON([]byte(in)); err != nil {
			t.Errorf("%q: upstream now refuses it (%v): remove it from the divergences", in, err)
		}
		if err := New[string, any]().UnmarshalJSON([]byte(in)); err == nil {
			t.Errorf("%q: json/v2 accepts it", in)
		}
		if err := json.Unmarshal([]byte(in), New[string, any]()); err == nil {
			t.Errorf("%q: encoding/json accepts it", in)
		}
	}
}

// BenchmarkJSONRoundTrip writes and reads a tool schema's properties, the
// shape eino-contrib/jsonschema keeps in ordered maps.
func BenchmarkJSONRoundTrip(b *testing.B) {
	om := New[string, any]()
	om.Set("checkIn", map[string]any{"type": "string", "format": "date", "description": "Data di arrivo"})
	om.Set("checkOut", map[string]any{"type": "string", "format": "date", "description": "Data di partenza"})
	om.Set("guests", map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"age": map[string]any{"type": "integer"}}}})
	om.Set("notes", map[string]any{"type": "string", "description": "Richieste particolari <vista mare>"})
	data, err := json.Marshal(om)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := json.Marshal(om); err != nil {
			b.Fatal(err)
		}
		if err := json.Unmarshal(data, New[string, any]()); err != nil {
			b.Fatal(err)
		}
	}
}
