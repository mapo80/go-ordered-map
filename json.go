package orderedmap

import (
	"encoding"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"unicode/utf8"
)

var (
	_ json.Marshaler         = &OrderedMap[int, any]{}
	_ json.Unmarshaler       = &OrderedMap[int, any]{}
	_ jsonv2.MarshalerTo     = &OrderedMap[int, any]{}
	_ jsonv2.UnmarshalerFrom = &OrderedMap[int, any]{}
)

// MarshalJSON implements the json.Marshaler interface, with the semantics of
// encoding/json for the values.
func (om *OrderedMap[K, V]) MarshalJSON() ([]byte, error) {
	return jsonv2.Marshal(om, json.DefaultOptionsV1())
}

// MarshalJSONTo implements the json/v2 MarshalerTo interface: the pairs in
// order, the values written with the options of enc.
func (om *OrderedMap[K, V]) MarshalJSONTo(enc *jsontext.Encoder) error {
	if om == nil || om.list == nil {
		return enc.WriteToken(jsontext.Null)
	}
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	var num jsontext.Value // an integer key as a JSON string, reused
	for pair := om.Oldest(); pair != nil; pair = pair.Next() {
		var err error
		switch key := any(pair.Key).(type) {
		case string:
			err = enc.WriteToken(jsontext.String(key))
		case encoding.TextMarshaler:
			var text []byte
			if text, err = key.MarshalText(); err == nil {
				err = enc.WriteToken(jsontext.String(string(text)))
			}
		default:
			// the kinds take care of wrapper types around primitive types, such as
			// type myType string
			switch keyValue := reflect.ValueOf(key); keyValue.Kind() {
			case reflect.String:
				err = enc.WriteToken(jsontext.String(keyValue.String()))
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				num = append(strconv.AppendInt(append(num[:0], '"'), keyValue.Int(), 10), '"')
				err = enc.WriteValue(num)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				num = append(strconv.AppendUint(append(num[:0], '"'), keyValue.Uint(), 10), '"')
				err = enc.WriteValue(num)
			default:
				return fmt.Errorf("unsupported key type: %T", key)
			}
		}
		if err != nil {
			return err
		}
		if err := jsonv2.MarshalEncode(enc, pair.Value); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndObject)
}

// UnmarshalJSON implements the json.Unmarshaler interface, with the semantics
// of encoding/json for the values.
func (om *OrderedMap[K, V]) UnmarshalJSON(data []byte) error {
	return jsonv2.Unmarshal(data, om, json.DefaultOptionsV1())
}

// errNotObject is reported when the JSON value is not an object.
var errNotObject = errors.New("orderedmap: JSON value is not an object")

// UnmarshalJSONFrom implements the json/v2 UnmarshalerFrom interface: the
// members are added in order (a repeated key keeps its first position and
// takes the last value), the values read with the options of dec.
func (om *OrderedMap[K, V]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if om.list == nil {
		om.initialize(0)
	}
	if kind := dec.PeekKind(); kind != '{' {
		if kind == 0 {
			_, err := dec.ReadToken()
			return err
		}
		if err := dec.SkipValue(); err != nil {
			return err
		}
		return errNotObject
	}
	if _, err := dec.ReadToken(); err != nil {
		return err
	}
	for dec.PeekKind() != '}' {
		raw, err := dec.ReadValue()
		if err != nil {
			return err
		}
		keyData, err := jsontext.AppendUnquote(nil, raw)
		if err != nil {
			return err
		}
		var key K
		switch typedKey := any(&key).(type) {
		case *string:
			if *typedKey, err = decodeUTF8(keyData); err != nil {
				return err
			}
		case encoding.TextUnmarshaler:
			if err := typedKey.UnmarshalText(keyData); err != nil {
				return err
			}
		case *int, *int8, *int16, *int32, *int64, *uint, *uint8, *uint16, *uint32, *uint64:
			if err := jsonv2.Unmarshal(keyData, typedKey, json.DefaultOptionsV1()); err != nil {
				return err
			}
		default:
			// the kinds take care of wrapper types around primitive types, such as
			// type myType string
			keyValue := reflect.ValueOf(&key).Elem()
			switch keyValue.Kind() {
			case reflect.String:
				s, err := decodeUTF8(keyData)
				if err != nil {
					return err
				}
				keyValue.SetString(s)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				// the key's text is read as a JSON number, as encoding/json does
				if err := jsonv2.Unmarshal(keyData, keyValue.Addr().Interface(), json.DefaultOptionsV1()); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported key type: %T", key)
			}
		}

		var value V
		if err := jsonv2.UnmarshalDecode(dec, &value); err != nil {
			return err
		}
		om.Set(key, value)
	}
	_, err := dec.ReadToken()
	return err
}

func decodeUTF8(input []byte) (string, error) {
	if !utf8.Valid(input) {
		for offset := 0; offset < len(input); {
			r, size := utf8.DecodeRune(input[offset:])
			if r == utf8.RuneError && size <= 1 {
				return "", fmt.Errorf("not a valid UTF-8 string (at position %d): %s", offset, string(input))
			}
			offset += size
		}
	}
	return string(input), nil
}
