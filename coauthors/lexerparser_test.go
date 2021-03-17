package coauthors

import (
	"testing"

	"mob.sh/testutils"
)

func TestLexJustOneAuthorWithoutExplicitAlias(t *testing.T) {
	input := `busta rhymes <>`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{tokAuthor, "busta rhymes <>"},
		{tokEol, ""},
	}

	l := newLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexJustOneCoauthorWithAlias(t *testing.T) {
	input := `Busta Rhymes <> as br`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{tokAuthor, "Busta Rhymes <>"},
		{tokAssign, "as"},
		{tokAlias, "br"},
		{tokEol, ""},
	}

	l := newLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexJustOneCoauthorOnlyAlias(t *testing.T) {
	input := `br`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{tokAlias, "br"},
		{tokEol, ""},
	}

	l := newLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexManyCoauthorsMixed(t *testing.T) {
	input := `abc, Cool Todd <todd@example.com> as todd, dhh,
			  Taylor Swift <> as t swizzle, Bond\, James <007@example.net> as jb,
	 		  AS <> as as, Janet Jackson <>, pencil neck`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{tokAlias, "abc"},
		{tokComma, ","},
		{tokAuthor, "Cool Todd <todd@example.com>"},
		{tokAssign, "as"},
		{tokAlias, "todd"},
		{tokComma, ","},
		{tokAlias, "dhh"},
		{tokComma, ","},
		{tokAuthor, "Taylor Swift <>"},
		{tokAssign, "as"},
		{tokAlias, "t swizzle"},
		{tokComma, ","},
		{tokAuthor, `Bond\, James <007@example.net>`},
		{tokAssign, "as"},
		{tokAlias, "jb"},
		{tokComma, ","},
		{tokAuthor, "AS <>"},
		{tokAssign, "as"},
		{tokAlias, "as"},
		{tokComma, ","},
		{tokAuthor, "Janet Jackson <>"},
		{tokComma, ","},
		{tokAlias, "pencil neck"},
		{tokEol, ""},
	}

	l := newLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestParseSingleCoauthorFullyQualified(t *testing.T) {
	input := `Daria Morgendorfer <daria@example.com> as dm`
	expected := map[Alias]Author{
		"dm": "Daria Morgendorfer <daria@example.com>",
	}

	l := newLexer(input)
	p := newParser(l)

	coauthors, _ := p.parseCoauthors()

	testutils.Equals(t, expected, coauthors)
}

func TestParseSingleCoauthorFullyQualifiedTrailingComma(t *testing.T) {
	input := `Daria Morgendorfer <daria@example.com> as dm,`
	expected := map[Alias]Author{
		"dm": "Daria Morgendorfer <daria@example.com>",
	}
	l := newLexer(input)
	p := newParser(l)

	coauthors, _ := p.parseCoauthors()

	testutils.Equals(t, expected, coauthors)
}
func TestParseManyCoauthorsFullyQualified(t *testing.T) {
	input := `Daria Morgendorfer <daria@example.com> as dm,
			  Taylor Swift <> as t swizzle, Bond\, James <007@example.net> as jb,
	 		  AS <> as as`

	expected := map[Alias]Author{
		"dm":        "Daria Morgendorfer <daria@example.com>",
		"t-swizzle": "Taylor Swift <>",
		"jb":        `Bond\, James <007@example.net>`,
		"as":        "AS <>",
	}

	l := newLexer(input)
	p := newParser(l)

	coauthors, _ := p.parseCoauthors()

	testutils.Equals(t, expected, coauthors)
}

func TestParseSingleCoauthorsNoAlias(t *testing.T) {
	input := `Janet Jackson <>`

	expected := map[Alias]Author{
		"jj": "Janet Jackson <>",
	}

	l := newLexer(input)
	p := newParser(l)

	coauthors, _ := p.parseCoauthors()

	testutils.Equals(t, expected, coauthors)
}

func TestParseManyCoauthorsMixed(t *testing.T) {
	input := `abc, Daria Morgendorfer <daria@example.com> as dm, dhh,
			  Taylor Swift <> as t swizzle, Bond\, James <007@example.net> as jb,
	 		  AS <> as as, Janet Jackson <>`

	expected := map[string]string{
		"abc":       "",
		"dm":        "Daria Morgendorfer <daria@example.com>",
		"dhh":       "",
		"t-swizzle": "Taylor Swift <>",
		"jb":        `Bond\, James <007@example.net>`,
		"as":        "AS <>",
		"jj":        "Janet Jackson <>",
	}

	l := newLexer(input)
	p := newParser(l)

	coauthors, _ := p.parseCoauthors()

	testutils.Equals(t, expected, coauthors)
}
