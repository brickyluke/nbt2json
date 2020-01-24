package nbt2json

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/ghodss/yaml"
)

// JsonParseError is when the data does not match an expected pattern. Pass it message string and downstream error
type JsonParseError struct {
	s string
	e error
}

func (e JsonParseError) Error() string {
	var s string
	if e.e != nil {
		s = fmt.Sprintf(": %s", e.e.Error())
	}
	return fmt.Sprintf("Error parsing json2nbt: %s%s", e.s, s)
}

// Yaml2Nbt converts JSON byte array to uncompressed NBT byte array
func Yaml2Nbt(b []byte, byteOrder binary.ByteOrder) ([]byte, error) {
	myJson, err := yaml.YAMLToJSON(b)
	if err != nil {
		return nil, JsonParseError{"Error converting YAML to JSON", err}
	}
	nbtOut, err := Json2Nbt(myJson, byteOrder)
	if err != nil {
		return nbtOut, err
	}
	return nbtOut, nil
}

// Json2Nbt converts JSON byte array to uncompressed NBT byte array
func Json2Nbt(b []byte, byteOrder binary.ByteOrder) ([]byte, error) {
	nbtOut := new(bytes.Buffer)
	var nbtJsonData NbtJson
	var nbtTag interface{}
	var nbtArray []interface{}
	var err error
	d := json.NewDecoder(bytes.NewBuffer(b))
	d.UseNumber() // keep number precision: float64, which is used by default cannot hold an int64
	err = d.Decode(&nbtJsonData)
	if err != nil {
		return nil, JsonParseError{"Error parsing JSON input. Is input JSON-formatted?", err}
	}
	temp, err := json.Marshal(nbtJsonData.Nbt)
	if err != nil {
		return nil, JsonParseError{"Error marshalling nbt: json.RawMessage", err}
	}
	d1 := json.NewDecoder(bytes.NewBuffer(temp))
	d1.UseNumber()
	err = d1.Decode(&nbtArray)
	if err != nil {
		return nil, JsonParseError{"Error unmarshalling nbt: value", err}
	}
	for _, nbtTag = range nbtArray {
		err = writeTag(nbtOut, byteOrder, nbtTag)
		if err != nil {
			return nil, err
		}
	}

	return nbtOut.Bytes(), nil
}

func writeTag(w io.Writer, byteOrder binary.ByteOrder, myMap interface{}) error {
	var err error
	if m, ok := myMap.(map[string]interface{}); ok {
		i64, err := m["tagType"].(json.Number).Int64()
		tagType := byte(i64)
		if err == nil {
			if tagType == 0 {
				// not expecting a 0 tag, but if it occurs just ignore it
				return nil
			}
			err = binary.Write(w, byteOrder, byte(tagType))
			if err != nil {
				return JsonParseError{"Error writing tagType" + string(tagType), err}
			}
			if name, ok := m["name"].(string); ok {
				err = binary.Write(w, byteOrder, int16(len(name)))
				if err != nil {
					return JsonParseError{"Error writing name length", err}
				}
				err = binary.Write(w, byteOrder, []byte(name))
				if err != nil {
					return JsonParseError{"Error converting name", err}
				}
			} else {
				return JsonParseError{"name field not a string", err}
			}
			err = writePayload(w, byteOrder, m, tagType)
			if err != nil {
				return err
			}

		} else {
			return JsonParseError{"tagType is not numeric", err}
		}
	} else {
		return JsonParseError{"writeTag: myMap is not map[string]interface{}", err}
	}
	return err
}

func writePayload(w io.Writer, byteOrder binary.ByteOrder, m map[string]interface{}, tagType byte) error {
	var err error

	switch tagType {
	case 1: // TAG_Byte
		i, err := m["value"].(json.Number).Int64()
		if err == nil {
			err = binary.Write(w, byteOrder, int8(i))
			if err != nil {
				return JsonParseError{"Error writing byte payload", err}
			}
		} else {
			return JsonParseError{"Tag Byte value field not a number", err}
		}
	case 2: // TAG_Short
		i, err := m["value"].(json.Number).Int64()
		if err == nil {
			err = binary.Write(w, byteOrder, int16(i))
			if err != nil {
				return JsonParseError{"Error writing short payload", err}
			}
		} else {
			return JsonParseError{"Tag Short value field not a number", err}
		}
	case 3: // TAG_Int
		i, err := m["value"].(json.Number).Int64()
		if err == nil {
			err = binary.Write(w, byteOrder, int32(i))
			if err != nil {
				return JsonParseError{"Error writing int32 payload", err}
			}
		} else {
			return JsonParseError{"Tag Int value field not a number", err}
		}
	case 4: // TAG_Long
		i, err := m["value"].(json.Number).Int64()
		if err == nil {
			err = binary.Write(w, byteOrder, int64(i))
			if err != nil {
				return JsonParseError{"Error writing int64 payload", err}
			}
		} else {
			return JsonParseError{"Tag Long value field not a number", err}
		}
	case 5: // TAG_Float
		f, err := m["value"].(json.Number).Float64()
		if err == nil {
			err = binary.Write(w, byteOrder, float32(f))
			if err != nil {
				return JsonParseError{"Error writing float32 payload", err}
			}
		} else {
			return JsonParseError{"Tag Float - Value field not a number", err}
		}
	case 6: // TAG_Double
		f, err := m["value"].(json.Number).Float64()
		if err == nil {
			err = binary.Write(w, byteOrder, f)
			if err != nil {
				return JsonParseError{"Tag Double - Error writing float64 payload", err}
			}
		} else {
			// return JsonParseError{"Tag Byte value field not a number", err}
			f = math.NaN()
			err = binary.Write(w, byteOrder, f)
			if err != nil {
				return JsonParseError{"Tag Double - Error writing float64 payload", err}
			}

		}
	case 7: // TAG_Byte_Array
		if values, ok := m["value"].([]interface{}); ok {
			err = binary.Write(w, byteOrder, int32(len(values)))
			if err != nil {
				return JsonParseError{"Error writing byte array length", err}
			}
			for _, value := range values {
				i, err := value.(json.Number).Int64()
				if err == nil {
					err = binary.Write(w, byteOrder, int8(i))
					if err != nil {
						return JsonParseError{"Error writing element of byte array", err}
					}
				} else {
					return JsonParseError{"Tag Byte Array - Tag Byte value field not a number", err}
				}
			}
		} else {
			return JsonParseError{"Tag Byte Array value field not an array", err}
		}
	case 8: // TAG_String
		if s, ok := m["value"].(string); ok {
			err = binary.Write(w, byteOrder, int16(len([]byte(s))))
			if err != nil {
				return JsonParseError{"Error writing string length", err}
			}
			err = binary.Write(w, byteOrder, []byte(s))
			if err != nil {
				return JsonParseError{"Error writing string payload", err}
			}
		} else {
			return JsonParseError{"Tag String value field not a number", err}
		}
	case 9: // TAG_List
		// important: tagListType needs to be in scope to be passed to writePayload
		// := were keeping it in a lower scope and zeroing it out.
		var tagListType byte
		if listMap, ok := m["value"].(map[string]interface{}); ok {
			i64, err := listMap["tagListType"].(json.Number).Int64()
			tagListType = byte(i64)
			if err == nil {
				err = binary.Write(w, byteOrder, byte(tagListType))
				if err != nil {
					return JsonParseError{"While writing tag list type", err}
				}
			}
			if values, ok := listMap["list"].([]interface{}); ok {
				err = binary.Write(w, byteOrder, int32(len(values)))
				if err != nil {
					return JsonParseError{"While writing tag list size", err}
				}
				for _, value := range values {
					fakeTag := make(map[string]interface{})
					fakeTag["value"] = value
					err = writePayload(w, byteOrder, fakeTag, tagListType)
					if err != nil {
						return JsonParseError{"While writing tag list of type " + strconv.Itoa(int(tagListType)), err}
					}
				}
			} else if listMap["list"] == nil {
				// NBT lists can be null / nil and therefore aren't represented as an array in JSON
				err = binary.Write(w, byteOrder, int32(0))
				if err != nil {
					return JsonParseError{"While writing tag list null size", err}
				}
				return nil
			} else {
				return JsonParseError{"Tag List's List value field not an array", err}
			}

		} else {
			return JsonParseError{"Tag List value field not an object", err}
		}
	case 10: // TAG_Compound
		if values, ok := m["value"].([]interface{}); ok {
			for _, value := range values {
				err = writeTag(w, byteOrder, value)
				if err != nil {
					return JsonParseError{"While writing Compound tags", err}
				}
			}
			// write the end tag which is just a single byte 0
			err = binary.Write(w, byteOrder, byte(0))
			if err != nil {
				return JsonParseError{"Writing End tag", err}
			}
		} else {
			return JsonParseError{"Tag Compound value field not an array", err}
		}
	case 11: // TAG_Int_Array
		if values, ok := m["value"].([]interface{}); ok {
			err = binary.Write(w, byteOrder, int32(len(values)))
			if err != nil {
				return JsonParseError{"Error writing int32 array length", err}
			}
			for _, value := range values {
				if i, ok := value.(float64); ok {
					err = binary.Write(w, byteOrder, int32(i))
					if err != nil {
						return JsonParseError{"Error writing element of int32 array", err}
					}
				} else {
					return JsonParseError{"Tag Int value field not a number", err}
				}
			}
		} else {
			return JsonParseError{"Tag Int Array value field not an array", err}
		}
	case 12: // TAG_Long_Array
		if values, ok := m["value"].([]interface{}); ok {
			err = binary.Write(w, byteOrder, int64(len(values)))
			if err != nil {
				return JsonParseError{"Error writing int64 array length", err}
			}
			for _, value := range values {
				if i, ok := value.(float64); ok {
					err = binary.Write(w, byteOrder, int64(i))
					if err != nil {
						return JsonParseError{"Error writing element of int64 array", err}
					}
				} else {
					return JsonParseError{"Tag Int value field not a number", err}
				}
			}
		} else {
			return JsonParseError{"Tag Int Array value field not an array", err}
		}
	default: // 0 = TAG_End
		return JsonParseError{"tagType " + string(tagType) + " is not recognized", err}
	}
	return err
}
