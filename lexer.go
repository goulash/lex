// Copyright (c) 2015, Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

// The code and API of this file is based on text/template/parse/lex.go
// from the Go standard library:
//
// Copyright (c) 2012 The Go Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
//      notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
//      copyright notice, this list of conditions and the following
//      disclaimer in the documentation and/or other materials provided
//      with the distribution.
//    * Neither the name of Google Inc. nor the names of its
//      contributors may be used to endorse or promote products derived
//      from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// Package lex provides a base lexer to help you implement your own.
//
// Feel free to copy this into your library (according to the license)
// as well as use it as a library. Both methods are possible.
//
// Code and API based off standard library text/template/parse.
package lex

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// EOF is returned by Lexer.Next upon reaching end-of-file.
const EOF = -1

// A Type is the type of a token.
//
// In this package, only several types are predefined. The rest you
// can set yourself like so:
//
//  const (
//      TypeSpace = (1+lex.TypeEOF)+iota // continue where lex left off
//      TypeNumber
//      TypeIdent
//      ...
//  )
//
type Type int

const (
	TypeError Type = iota // string is error text
	TypeEOF               // end-of-file, last reserved type
)

type Token struct {
	Type
	Pos   int
	Value string
}

type StateFn func(*Lexer) StateFn

type Lexer struct {
	name    string
	input   string
	width   int
	base    int
	pos     int
	lastPos int
	tokens  chan Token
}

// New creates a new Lexer and returns it.
//
// Before calling NextToken, it should be run in a separate goroutine:
//
//  l := lex.New(name, input)
//  go l.Run(sf)
//  ...
//  t := l.NextToken()
func New(name, input string) *Lexer {
	l := &Lexer{
		name:   name,
		input:  input,
		tokens: make(chan Token),
	}
	return l
}

// Lex creates a new Lexer and starts running it with sf.
func Lex(name, input string, sf StateFn) *Lexer {
	l := New(name, input)
	go l.Run(sf)
	return l
}

// Run starts the lexer with the given StateFn.
// After receiving a nil StateFn, it closes the tokens channel.
func (l *Lexer) Run(fn StateFn) {
	for state := fn; state != nil; {
		state = state(l)
	}
	close(l.tokens)
}

// NextToken returns the next token from the input.
// Called by the parser, not in the lexing goroutine.
//
// Note: if l.Run has not been called, NextToken will block.
func (l *Lexer) NextToken() Token {
	t := <-l.tokens
	l.lastPos = t.Pos
	return t
}

// Drain drains the output so the lexing goroutine will exit.
// Called by the parser, not in the lexing goroutine.
func (l *Lexer) Drain() {
	for range l.tokens {
	}
}

// LineNumber reports the line of the last token returned by NextToken.
func (l *Lexer) LineNumber() int {
	return 1 + strings.Count(l.input[:l.lastPos], "\n")
}

// ColumnNumber reports the column of the last token returned by NextToken.
func (l *Lexer) ColumnNumber() int {
	code := l.input[:l.lastPos]
	if i := strings.LastIndex(code, "\n"); i >= 0 {
		return l.lastPos - i
	} else {
		return 1 + len(code)
	}
}

// Name returns the name of the input.
func (l *Lexer) Name() string { return l.name }

// Value returns the current token value, essentially the part of input
// from l.base to l.pos.
func (l *Lexer) Value() string { return l.input[l.base:l.pos] }

// Len returns the size of the current read token.
func (l *Lexer) Len() int { return l.base - l.pos }

// Inc increments the position by n.
func (l *Lexer) Inc(n int) { l.pos += n }

// Dec decrements the position by n.
func (l *Lexer) Dec(n int) { l.pos -= n }

// Input returns a slice of the current position plus n.
func (l *Lexer) Input(n int) string {
	return l.input[l.pos+n:]
}

// Next returns the next rune in the input.
// If there is no more input left to read, EOF is returned.
func (l *Lexer) Next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return EOF
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += l.width
	return r
}

// Peek returns but does not consume the next rune in the input.
func (l *Lexer) Peek() rune {
	r := l.Next()
	l.Backup()
	return r
}

// Emit passes a token back to the client.
func (l *Lexer) Emit(t Type) {
	l.tokens <- Token{t, l.base, l.input[l.base:l.pos]}
	l.base = l.pos
}

// Ignore skips over the pending input before this point.
func (l *Lexer) Ignore() {
	l.base = l.pos
}

// Backup steps back one rune. Can only be called once per call of Next.
func (l *Lexer) Backup() {
	l.pos -= l.width
}

// Consume tries to consume exactly the string s.
func (l *Lexer) Consume(s string) bool {
	ok := l.HasPrefix(s)
	if ok {
		l.pos += len(s)
	}
	return ok
}

// Accept consumes the next rune if it is from the valid set.
func (l *Lexer) Accept(valid string) bool {
	if strings.IndexRune(valid, l.Next()) >= 0 {
		return true
	}
	l.Backup()
	return false
}

// AcceptRun consumes a run of runes from the valid set.
// The number of bytes advanced is returned.
func (l *Lexer) AcceptRun(valid string) int {
	var n int
	for strings.IndexRune(valid, l.Next()) >= 0 {
		n += l.width
	}
	l.Backup()
	return n
}

// AcceptFunc consumes the next rune if f returns true.
func (l *Lexer) AcceptFunc(f func(r rune) bool) bool {
	if f(l.Next()) {
		return true
	}
	l.Backup()
	return false
}

// AcceptFunc consumes a run of runes as long as f returns true.
// The number of bytes advanced is returned.
func (l *Lexer) AcceptFuncRun(f func(r rune) bool) int {
	var n int
	for f(l.Next()) {
		n += l.width
	}
	l.Backup()
	return n
}

// AcceptBut consumes a rune if it is not from the invalid set.
func (l *Lexer) AcceptBut(invalid string) bool {
	if strings.IndexRune(invalid, l.Next()) < 0 {
		return true
	}
	l.Backup()
	return false
}

// AcceptButRun consumes runes as long as they are not in the invalid set.
// The number of bytes advanced is returned.
func (l *Lexer) AcceptButRun(invalid string) int {
	var n int
	for strings.IndexRune(invalid, l.Next()) < 0 {
		n += l.width
	}
	l.Backup()
	return n
}

// HasPrefix returns true if the input from the current position
// has the prefix s. It does not consume the prefix.
func (l *Lexer) HasPrefix(s string) bool {
	return strings.HasPrefix(l.input[l.pos:], s)
}

// HasPrefixAfter returns true if the input from the current position
// plus after bytes has the prefix s. It does not consume the prefix.
func (l *Lexer) HasPrefixAfter(after int, s string) bool {
	return strings.HasPrefix(l.input[l.pos+after:], s)
}

// Errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.NextToken.
func (l *Lexer) Errorf(format string, args ...interface{}) StateFn {
	l.tokens <- Token{TypeError, l.base, fmt.Sprintf(format, args...)}
	return nil
}
