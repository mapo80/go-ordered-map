# Fork mapo80/go-ordered-map

Fork of `github.com/wk8/go-ordered-map/v2` for ARC (`arc-workspace`). The module path is unchanged; ARC uses it with `replace github.com/wk8/go-ordered-map/v2 => github.com/mapo80/go-ordered-map/v2 <tag>`.

- **Upstream base:** tag `v2.1.8`, commit `85ca4a2b29d3241fa4513f82be3d38fe2392a791`.
- **License:** unchanged (`LICENSE`, Apache 2.0 as upstream).
- **Branch of the changes:** `arc/json-v2`, recorded in `master` with a merge that keeps master's content (`master` follows upstream past v2.1.8). ARC uses the annotated tag `v2.1.8-arc.1` on `arc/json-v2`.
- **Go:** `go 1.27` (`encoding/json/v2` and `encoding/json/jsontext` from the standard library).
- **Goal:** one JSON library in ARC's server build, the standard library's `encoding/json/v2`.

## Changes

| commit | change |
|---|---|
| `eae5fa9` | `OrderedMap` implements `json/v2` `MarshalerTo` and `UnmarshalerFrom`: keys and values go to and from the caller's `jsontext` stream (values with the caller's options, so `encoding/json` callers get its semantics); `MarshalJSON`/`UnmarshalJSON` run the same code with `encoding/json`'s options (`DefaultOptionsV1`). Repeated keys keep the first position and the last value; string keys with invalid UTF-8 are refused on read as before; integer keys are read as JSON numbers as before. `buger/jsonparser` and `mailru/easyjson` are no longer in the build of the package; `json.go` of v2.1.8 is kept as the test reference (`json_reference_test.go`). |

## Equivalence

- `TestMarshalMatchesUpstream`: maps with string, int, uint64, int16, named string and `TextMarshaler` keys; values with HTML characters, U+2028/2029, control characters, invalid UTF-8, nil slices and maps, null, int64/uint64 limits, floats, `json.Number`, `json.RawMessage`, bytes, structs, nested maps and nil maps. Same bytes as upstream through `MarshalJSON`, `encoding/json` (alone and inside a struct) and `json/v2` with the `encoding/json` options.
- `TestUnmarshalMatchesUpstream`: 32 inputs (duplicates, exact and out-of-range numbers, escapes, nul, invalid UTF-8 in keys and values, lone surrogates, non-objects, malformed input, integer and `TextUnmarshaler` keys) into seven key/value type pairs, directly and through `encoding/json`: same pairs in the same order or an error in both.
- Known differences, both tested:
  - a string **key** with invalid UTF-8 is written as U+FFFD (as `encoding/json` writes it), where easyjson wrote the escape `�`; both read back as the same pairs (`TestInvalidUTF8KeyReadsTheSame`);
  - `UnmarshalJSON` called directly with bytes that are not one JSON object (`{"a":1,}`, `{"a":1}}`, two objects) now fails; jsonparser accepted them. Through `encoding/json` both fail, since it validates the input first (`TestUnmarshalDivergencesAreOnlyMalformedInput`).
- Upstream tests green; `FuzzRoundTripJSON` 20 s (950 000 executions) without failures.

## Measures (8 November 2026, Apple M2 Pro, go1.27.1, benchstat n=10)

| benchmark | upstream | fork | |
|---|---|---|---|
| `BenchmarkMarshalJSON` (upstream benchmark) time/op | 2.12 µs | 1.18 µs | −44 % |
| `BenchmarkJSONRoundTrip` (tool schema properties, write + read) bytes/op | 6 112 | 4 811 | −21 % |
| `BenchmarkJSONRoundTrip` allocs/op | 130 | 124 | −5 % |
| `BenchmarkJSONRoundTrip` time/op | 18.9 µs | 19.5 µs | ~ (p=0.85) |
