// Package pipeenc implements encoding and decoding of the serialization format
// used by trqauthd.
package pipeenc

import (
	"errors"
	"strconv"
	"strings"
)

const (
	delimiter = "|"
)

var (
	errUnexpectedEnd = errors.New("unexpeced end of string")
	errBadFormat     = errors.New("bad format")
)

// Encoder holds a buffer for pipe encoding.
type Encoder struct {
	buf string
}

// NewEncoder returns an Encoder.
func NewEncoder() Encoder {
	return Encoder{}
}

// PutInt pipe-encodes an integer and appends the result to the buffer.
func (enc *Encoder) PutInt(i int) {
	enc.buf += strconv.Itoa(i)
	enc.buf += delimiter
}

// PutString pipe-encodes a string and appends the result to the buffer.
func (enc *Encoder) PutString(s string) {
	enc.PutInt(len(s))
	enc.buf += s
	enc.buf += delimiter
}

// String returns the constructed string.
func (enc *Encoder) String() string {
	return enc.buf
}

// Decoder holds a buffer for pipe decoding.
type Decoder struct {
	buf string
}

// NewDecoder returns a Decoder that decodes s.
func NewDecoder(s string) Decoder {
	return Decoder{buf: s}
}

// GetInt pipe-decodes an integer from the buffer.
func (dec *Decoder) GetInt() (int, error) {
	pos := strings.Index(dec.buf, delimiter)
	if pos == -1 {
		return 0, errBadFormat
	}

	num, err := strconv.Atoi(dec.buf[:pos])
	if err != nil {
		return 0, err
	}

	dec.buf = dec.buf[pos+1:]

	return num, nil
}

// GetString pipe-decodes a string from the buffer.
func (dec *Decoder) GetString() (string, error) {
	n, err := dec.GetInt()
	if err != nil {
		return "", err
	}

	if len(dec.buf) < n+1 {
		return "", errUnexpectedEnd
	}

	if !strings.HasPrefix(dec.buf[n:], delimiter) {
		return "", errBadFormat
	}

	s := dec.buf[:n]
	dec.buf = dec.buf[n+1:]

	return s, nil
}
