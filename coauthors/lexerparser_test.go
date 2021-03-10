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
	input := `abc, Butt Head <notbeavis@mtv.net> as bh, dhh, 
			  Taylor Swift <> as t swizzle, Bond\, James <007@mi6.co.uk> as jb,
	 		  AS <> as as, Janet Jackson <>, pencil neck`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{tokAlias, "abc"},
		{tokComma, ","},
		{tokAuthor, "Butt Head <notbeavis@mtv.net>"},
		{tokAssign, "as"},
		{tokAlias, "bh"},
		{tokComma, ","},
		{tokAlias, "dhh"},
		{tokComma, ","},
		{tokAuthor, "Taylor Swift <>"},
		{tokAssign, "as"},
		{tokAlias, "t swizzle"},
		{tokComma, ","},
		{tokAuthor, `Bond\, James <007@mi6.co.uk>`},
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
	input := `Butt Head <notbeavis@mtv.net> as bh`
	expected := map[Alias]Author{
		"bh": "Butt Head <notbeavis@mtv.net>",
	}

	l := newLexer(input)
	p := newParser(l)

	coauthors, _ := p.parseCoauthors()

	testutils.Equals(t, expected, coauthors)
}

func TestParseSingleCoauthorFullyQualifiedTrailingComma(t *testing.T) {
	input := `Butt Head <notbeavis@mtv.net> as bh,`
	expected := map[Alias]Author{
		"bh": "Butt Head <notbeavis@mtv.net>",
	}
	l := newLexer(input)
	p := newParser(l)

	coauthors, _ := p.parseCoauthors()

	testutils.Equals(t, expected, coauthors)
}
func TestParseManyCoauthorsFullyQualified(t *testing.T) {
	input := `Butt Head <notbeavis@mtv.net> as bh,
			  Taylor Swift <> as t swizzle, Bond\, James <007@mi6.co.uk> as jb,
	 		  AS <> as as`

	expected := map[Alias]Author{
		"bh":        "Butt Head <notbeavis@mtv.net>",
		"t-swizzle": "Taylor Swift <>",
		"jb":        `Bond\, James <007@mi6.co.uk>`,
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
	input := `abc, Butt Head <notbeavis@mtv.net> as bh, dhh,
			  Taylor Swift <> as t swizzle, Bond\, James <007@mi6.co.uk> as jb,
	 		  AS <> as as, Janet Jackson <>`

	expected := map[string]string{
		"abc":       "",
		"bh":        "Butt Head <notbeavis@mtv.net>",
		"dhh":       "",
		"t-swizzle": "Taylor Swift <>",
		"jb":        `Bond\, James <007@mi6.co.uk>`,
		"as":        "AS <>",
		"jj":        "Janet Jackson <>",
	}

	l := newLexer(input)
	p := newParser(l)

	coauthors, _ := p.parseCoauthors()

	testutils.Equals(t, expected, coauthors)
}
