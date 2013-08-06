package nbt

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	TAG_End = iota
	TAG_Byte
	TAG_Short
	TAG_Int
	TAG_Long
	TAG_Float
	TAG_Double
	TAG_Byte_Array
	TAG_String
	TAG_List
	TAG_Compound
	TAG_Int_Array
)

type TagType byte

func (tt TagType) String() string {
	switch tt {
	case TAG_End:
		return "TAG_End"
	case TAG_Byte:
		return "TAG_Byte"
	case TAG_Short:
		return "TAG_Short"
	case TAG_Int:
		return "TAG_Int"
	case TAG_Long:
		return "TAG_Long"
	case TAG_Float:
		return "TAG_Float"
	case TAG_Double:
		return "TAG_Double"
	case TAG_Byte_Array:
		return "TAG_Byte_Array"
	case TAG_String:
		return "TAG_String"
	case TAG_List:
		return "TAG_List"
	case TAG_Compound:
		return "TAG_Compound"
	case TAG_Int_Array:
		return "TAG_Int_Array"
	default:
		return "TAG_Unknown"
	}
}

type Tag struct {
	Type    TagType
	Payload interface{}
}

type TagList struct {
	Type  TagType
	Elems []interface{}
}

type TagCompound map[string]Tag

func readByte(r io.Reader) (byte, error) {
	buf := make([]byte, 1)
	_, err := r.Read(buf)
	return buf[0], err
}

func readTagData(r io.Reader, tt TagType) (interface{}, error) {
	switch tt {
	case TAG_End:
	case TAG_Byte:
		var v uint8
		err := binary.Read(r, binary.BigEndian, &v)
		return v, err
	case TAG_Short:
		var v int16
		err := binary.Read(r, binary.BigEndian, &v)
		return v, err
	case TAG_Int:
		var v int32
		err := binary.Read(r, binary.BigEndian, &v)
		return v, err
	case TAG_Long:
		var v int64
		err := binary.Read(r, binary.BigEndian, &v)
		return v, err
	case TAG_Float:
		var v float32
		err := binary.Read(r, binary.BigEndian, &v)
		return v, err
	case TAG_Double:
		var v float64
		err := binary.Read(r, binary.BigEndian, &v)
		return v, err
	case TAG_Byte_Array:
		var l int32
		if err := binary.Read(r, binary.BigEndian, &l); err != nil {
			return nil, err
		}
		if l < 0 {
			return nil, errors.New("Byte array has negative length?")
		}

		data := make([]byte, l)
		_, err := io.ReadFull(r, data)
		return data, err
	case TAG_String:
		var l int16
		if err := binary.Read(r, binary.BigEndian, &l); err != nil {
			return nil, err
		}
		if l < 0 {
			return nil, errors.New("String has negative length?")
		}

		data := make([]byte, l)
		_, err := io.ReadFull(r, data)
		return string(data), err
	case TAG_List:
		ltt, err := readByte(r)
		if err != nil {
			return nil, err
		}
		ltt := TagType(_ltt)

		var l int32
		if err := binary.Read(r, binary.BigEndian, &l); err != nil {
			return nil, err
		}
		if l < 0 {
			return nil, errors.New("List has negative length?")
		}

		tl := TagList{Type: ltt, Elems: make([]interface{}, l)}
		for i := 0; i < int(l); i++ {
			if tl.Elems[i], err = readTagData(r, ltt); err != nil {
				return nil, err
			}
		}
		return tl, nil
	case TAG_Compound:
		comp := make(TagCompound)
		for {
			tag, name, err := ReadNamedTag(r)
			if err != nil {
				return nil, err
			}
			if tag.Type == TAG_End {
				break
			}
			comp[name] = tag
		}
		return comp, nil
	case TAG_Int_Array:
		var l int32
		if err := binary.Read(r, binary.BigEndian, &l); err != nil {
			return nil, err
		}
		if l < 0 {
			return nil, errors.New("Int Array has negative length?")
		}

		data := make([]int32, l)
		for i := 0; i < int(l); i++ {
			var e int32
			if err := binary.Read(r, binary.BigEndian, &e); err != nil {
				return nil, err
			}
			data[i] = e
		}
		return data, nil
	}

	return nil, errors.New("Unknown tag type")
}

func ReadNamedTag(r io.Reader) (Tag, string, error) {
	_tt, err := kagus.ReadByte(r)
	if err != nil {
		return Tag{}, "", err
	}
	tt := TagType(_tt)

	if tt == TAG_End {
		return Tag{Type: tt}, "", nil
	}

	name, err := readTagData(r, TAG_String)
	if err != nil {
		return Tag{}, "", err
	}

	td, err := readTagData(r, tt)
	return Tag{Type: tt, Payload: td}, name.(string), err
}

func writeByte(w io.Writer, b byte) error {
	_, err := w.Write([]byte{b})
	return err
}

func writeTagData(w io.Writer, tt TagType, data interface{}) error {
	switch tt {
	case TAG_End:
		return nil
	case TAG_Byte:
		return writeByte(w, data.(byte))
	case TAG_Short:
		return binary.Write(w, binary.BigEndian, data.(int16))
	case TAG_Int:
		return binary.Write(w, binary.BigEndian, data.(int32))
	case TAG_Long:
		return binary.Write(w, binary.BigEndian, data.(int64))
	case TAG_Float:
		return binary.Write(w, binary.BigEndian, data.(float32))
	case TAG_Double:
		return binary.Write(w, binary.BigEndian, data.(float64))
	case TAG_Byte_Array:
		slice := data.([]byte)
		if err := binary.Write(w, binary.BigEndian, int32(len(slice))); err != nil {
			return err
		}
		_, err := w.Write(slice)
		return err
	case TAG_String:
		strEnc := []byte(data.(string))
		if err := binary.Write(w, binary.BigEndian, int16(len(strEnc))); err != nil {
			return err
		}
		_, err := w.Write(strEnc)
		return err
	case TAG_List:
		list := data.(TagList)
		if err := writeByte(w, byte(list.Type)); err != nil {
			return err
		}

		if err := binary.Write(w, binary.BigEndian, int32(len(list.Elems))); err != nil {
			return err
		}

		for _, el := range list.Elems {
			if err := writeTagData(w, list.Type, el); err != nil {
				return err
			}
		}
		return nil
	case TAG_Compound:
		comp := data.(TagCompound)
		for name, tag := range comp {
			if err := WriteNamedTag(w, name, tag); err != nil {
				return err
			}
		}
		return writeByte(w, TAG_End)
	case TAG_Int_Array:
		slice := data.([]int32)
		if err := binary.Write(w, binary.BigEndian, int32(len(slice))); err != nil {
			return err
		}

		for _, el := range slice {
			if err := binary.Write(w, binary.BigEndian, el); err != nil {
				return err
			}
		}

		return nil
	}

	return errors.New("Unknown tage type")
}

func WriteNamedTag(w io.Writer, name string, tag Tag) error {
	if err := writeByte(w, byte(tag.Type)); err != nil {
		return err
	}

	if err := writeTagData(w, TAG_String, name); err != nil {
		return err
	}

	return writeTagData(w, tag.Type, tag.Payload)
}