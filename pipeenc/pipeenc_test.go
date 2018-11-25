package pipeenc

import (
	"testing"
)

func Test_Encoder_PutInt(t *testing.T) {
	enc := NewEncoder()
	enc.PutInt(1)
	enc.PutInt(23)
	enc.PutInt(456)

	actual := enc.String()
	expected := "1|23|456|"

	if actual != expected {
		t.Errorf("unexpected result: got %q, want %q", actual, expected)
	}
}

func Test_Encoder_PutString(t *testing.T) {
	enc := NewEncoder()
	enc.PutString("Lorem")
	enc.PutString("ipsum")
	enc.PutString("dolor sit amet")

	actual := enc.String()
	expected := "5|Lorem|5|ipsum|14|dolor sit amet|"

	if actual != expected {
		t.Errorf("unexpected result: got %q, want %q", actual, expected)
	}
}

func Test_Decoder_GetInt(t *testing.T) {
	dec := NewDecoder("1|23|456|")
	expects := []int{1, 23, 456}

	for _, expected := range expects {
		actual, err := dec.GetInt()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if actual != expected {
			t.Errorf("unexpected result: got %d, want %d", actual, expected)
		}
	}
}

func Test_Decoder_GetInt_rejectsBadInput(t *testing.T) {
	badCases := []string{
		"|",
		"12a|",
		"1234", // no delimiter
	}

	for _, badCase := range badCases {
		dec := NewDecoder(badCase)
		actual, err := dec.GetInt()

		if err == nil {
			t.Errorf("unexpected success: got %d", actual)
		}
	}
}

func Test_Decoder_GetString(t *testing.T) {
	dec := NewDecoder("5|Lorem|5|ipsum|14|dolor sit amet|")
	expects := []string{"Lorem", "ipsum", "dolor sit amet"}

	for _, expected := range expects {
		actual, err := dec.GetString()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if actual != expected {
			t.Errorf("unexpected result: got %q, want %q", actual, expected)
		}
	}
}

func Test_Decoder_GetString_RejectsBadInput(t *testing.T) {
	badCases := []string{
		"|",
		"12a|",
		"1234",            // no delimiter
		"10|Lorem ipsum|", // length mismatch
	}

	for _, badCase := range badCases {
		dec := NewDecoder(badCase)
		actual, err := dec.GetString()

		if err == nil {
			t.Errorf("unexpected success: got %q", actual)
		}
	}
}
