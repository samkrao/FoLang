package scanlex

import "testing"

func TestTokenStreamPeekBuffersOnlyRequestedLookahead(t *testing.T) {
	const source = "alpha beta gamma delta"
	stream := NewTokenStream([]byte(source), "test.fol")

	if got := stream.Peek(0); got.Value != "alpha" {
		t.Fatalf("Peek(0) = %q, want alpha", got.Value)
	}
	if got := len(stream.buffer); got != 1 {
		t.Fatalf("Peek(0) buffered %d tokens, want 1", got)
	}
	if got := stream.lexer.inner.pos; got != len("alpha") {
		t.Fatalf("Peek(0) scanned through byte %d, want %d", got, len("alpha"))
	}

	if got := stream.Peek(2); got.Value != "gamma" {
		t.Fatalf("Peek(2) = %q, want gamma", got.Value)
	}
	if got := len(stream.buffer); got != 3 {
		t.Fatalf("Peek(2) buffered %d tokens, want 3", got)
	}
	if got := stream.lexer.inner.pos; got >= len(source) {
		t.Fatalf("Peek(2) eagerly scanned the complete source through byte %d", got)
	}
}

func TestTokenStreamNextBuffersOneToken(t *testing.T) {
	stream := NewTokenStream([]byte("alpha beta"), "test.fol")

	if got := stream.Next(); got.Value != "alpha" {
		t.Fatalf("first Next() = %q, want alpha", got.Value)
	}
	if len(stream.buffer) != 0 {
		t.Fatalf("Next() retained %d unrequested tokens", len(stream.buffer))
	}
	if got := stream.lexer.inner.pos; got != len("alpha") {
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
	if got := len(stream.history); got != 2 {
		t.Fatalf("EOF calls changed consumed history length to %d, want 2", got)
	}
}

func TestTokenStreamPeekPastEOFStopsScanning(t *testing.T) {
	stream := NewTokenStream([]byte("only"), "test.fol")

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

func TestTokenStreamPreviousTracksOnlyConsumedTokens(t *testing.T) {
	stream := NewTokenStream([]byte("alpha beta gamma"), "test.fol")

	if _, ok := stream.Previous(1); ok {
		t.Fatal("Previous(1) succeeded before a token was consumed")
	}
	if got := stream.Peek(2); got.Value != "gamma" {
		t.Fatalf("Peek(2) = %q, want gamma", got.Value)
	}
	if _, ok := stream.Previous(1); ok {
		t.Fatal("Peek unexpectedly added a token to consumed history")
	}

	if got := stream.Next(); got.Value != "alpha" {
		t.Fatalf("first Next() = %q, want alpha", got.Value)
	}
	if got, ok := stream.Previous(1); !ok || got.Value != "alpha" {
		t.Fatalf("Previous(1) = (%q, %v), want (alpha, true)", got.Value, ok)
	}
	if _, ok := stream.Previous(2); ok {
		t.Fatal("Previous(2) succeeded with only one consumed token")
	}

	if got := stream.Next(); got.Value != "beta" {
		t.Fatalf("second Next() = %q, want beta", got.Value)
	}
	if got, ok := stream.Previous(1); !ok || got.Value != "beta" {
		t.Fatalf("Previous(1) = (%q, %v), want (beta, true)", got.Value, ok)
	}
	if got, ok := stream.Previous(2); !ok || got.Value != "alpha" {
		t.Fatalf("Previous(2) = (%q, %v), want (alpha, true)", got.Value, ok)
	}
}

func TestTokenStreamPreviousRejectsNonPositiveLookbehind(t *testing.T) {
	stream := NewTokenStream(nil, "test.fol")
	defer func() {
		if recover() == nil {
			t.Fatal("Previous(0) did not panic")
		}
	}()
	stream.Previous(0)
}
