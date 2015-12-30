// Copyright (c) 2015, Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package lex

import "unicode"

var (
	Space   = " \t"
	Endline = "\r\n"
	Quote   = "\"'`"
)

func IsSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

func IsEndline(r rune) bool {
	return r == '\r' || r == '\n'
}

func IsAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func IsQuote(r rune) bool {
	return r == '"' || r == '\'' || r == '`'
}
