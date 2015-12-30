// Copyright (c) 2015, Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package lex

type Reader struct {
	lex *Lexer
	buf *Token
}

func NewReader(l *Lexer) *Reader {
	return &Reader{lex: l}
}

func (r *Reader) Peek() Token {
	if r.buf == nil {
		t := r.lex.NextToken()
		r.buf = &t
	}
	return *r.buf
}

func (r *Reader) Next() Token {
	if r.buf != nil {
		t := r.buf
		r.buf = nil
		return *t
	}
	return r.lex.NextToken()
}

func (r *Reader) Backup(t Token) {
	if r.buf != nil {
		panic("cannot backup more than one token")
	}
	r.buf = &t
}

func (r *Reader) PosInfo() (name string, line, col int) {
	return r.lex.Name(), r.lex.LineNumber(), r.lex.ColumnNumber()
}
