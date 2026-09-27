package scanlex

import "testing"

func TestTokenStreamPeekBuffersOnlyRequestedLookahead(t *testing.T) {
	const source = "alpha beta gamma delta"
	lexer := NewLexer([]byte(source), "test.fol", nil)
	stream := NewTokenStream(lexer)

	if got := stream.Peek(0); got.Value != "alpha" {
		t.Fatalf("Peek(0) = %q, want alpha", got.Value)
	}
	if got := len(stream.buffer); got != 1 {
		t.Fatalf("Peek(0) buffered %d tokens, want 1", got)
	}
	if got := lexer.inner.pos; got != len("alpha") {
		t.Fatalf("Peek(0) scanned through byte %d, want %d", got, len("alpha"))
	}

	if got := stream.Peek(2); got.Value != "gamma" {
		t.Fatalf("Peek(2) = %q, want gamma", got.Value)
	}
	if got := len(stream.buffer); got != 3 {
		t.Fatalf("Peek(2) buffered %d tokens, want 3", got)
	}
	if got := lexer.inner.pos; got >= len(source) {
		t.Fatalf("Peek(2) eagerly scanned the complete source through byte %d", got)
	}
}

func TestTokenStreamNextBuffersOneToken(t *testing.T) {
	lexer := NewLexer([]byte("alpha beta"), "test.fol", nil)
	stream := NewTokenStream(lexer)

	if got := stream.Next(); got.Value != "alpha" {
		t.Fatalf("first Next() = %q, want alpha", got.Value)
	}
	if len(stream.buffer) != 0 {
		t.Fatalf("Next() retained %d unrequested tokens", len(stream.buffer))
	}
	if got := lexer.inner.pos; got != len("alpha") {
		t.Fatalf("Next() scanned through byte %d, want %d", got, len("alpha"))
	}

	if got := stream.Next(); got.Value != "beta" {
		t.Fatalf("second Next() = %q, want beta", got.Value)
	}
	if got := stream.Next(); got.Kind != EOF {
		t.Fatalf("third Next() kind = %v, want EOF", got.Kind)
	}
	if got := stream.Next(); got.Kind != EOF {
		t.Fatalf("EOF was not stable: subsequent kind = %v", got.Kind)
	}
}

func TestTokenStreamPeekPastEOFStopsScanning(t *testing.T) {
	lexer := NewLexer([]byte("only"), "test.fol", nil)
	stream := NewTokenStream(lexer)

	if got := stream.Peek(10); got.Kind != EOF {
		t.Fatalf("Peek past EOF kind = %v, want EOF", got.Kind)
	}
	if !stream.exhausted {
		t.Fatal("stream did not remember reaching EOF")
	}
	if got := len(stream.buffer); got != 1 {
		t.Fatalf("stream buffered %d source tokens, want 1", got)
	}
	if got := stream.Peek(10); got.Kind != EOF {
		t.Fatalf("repeated Peek past EOF kind = %v, want EOF", got.Kind)
	}
}
