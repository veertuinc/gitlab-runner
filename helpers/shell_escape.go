package helpers

// https://github.com/zimbatm/direnv/blob/master/shell.go

import (
	"bytes"
	"encoding/hex"
	"strings"
)

/*
 * Escaping
 */

const (
	ACK           = 6
	TAB           = 9
	LF            = 10
	CR            = 13
	US            = 31
	SPACE         = 32
	AMPERSTAND    = 38
	SINGLE_QUOTE  = 39
	PLUS          = 43
	NINE          = 57
	QUESTION      = 63
	LOWERCASE_Z   = 90
	OPEN_BRACKET  = 91
	BACKSLASH     = 92
	UNDERSCORE    = 95
	CLOSE_BRACKET = 93
	BACKTICK      = 96
	TILDA         = 126
	DEL           = 127
)

// ShellEscape is taken from https://github.com/solidsnack/shell-escape/blob/master/Text/ShellEscape/Bash.hs
/*
A Bash escaped string. The strings are wrapped in @$\'...\'@ if any
bytes within them must be escaped; otherwise, they are left as is.
Newlines and other control characters are represented as ANSI escape
sequences. High bytes are represented as hex codes. Thus Bash escaped
strings will always fit on one line and never contain non-ASCII bytes.
*/
func ShellEscape(str string) string {
	if str == "" {
		return "''"
	}
	in := []byte(str)
	out := bytes.NewBuffer(make([]byte, 0, len(str)*2))
	i := 0
	l := len(in)
	escape := false

	hex := func(char byte) {
		escape = true

		data := []byte{BACKSLASH, 'x', 0, 0}
		hex.Encode(data[2:], []byte{char})
		out.Write(data)
	}

	backslash := func(char byte) {
		escape = true
		out.Write([]byte{BACKSLASH, char})
	}

	escaped := func(str string) {
		escape = true
		out.WriteString(str)
	}

	quoted := func(char byte) {
		escape = true
		out.WriteByte(char)
	}

	literal := func(char byte) {
		out.WriteByte(char)
	}

	for i < l {
		char := in[i]
		switch {
		case char == TAB:
			escaped(`\t`)
		case char == LF:
			escaped(`\n`)
		case char == CR:
			escaped(`\r`)
		case char <= US:
			hex(char)
		case char <= AMPERSTAND:
			quoted(char)
		case char == SINGLE_QUOTE:
			backslash(char)
		case char <= PLUS:
			quoted(char)
		case char <= NINE:
			literal(char)
		case char <= QUESTION:
			quoted(char)
		case char <= LOWERCASE_Z:
			literal(char)
		case char == OPEN_BRACKET:
			quoted(char)
		case char == BACKSLASH:
			backslash(char)
		case char <= CLOSE_BRACKET:
			quoted(char)
		case char == UNDERSCORE:
			literal(char)
		case char <= BACKTICK:
			quoted(char)
		case char <= TILDA:
			quoted(char)
		case char == DEL:
			hex(char)
		default:
			hex(char)
		}
		i++
	}

	outStr := out.String()

	if escape {
		outStr = "$'" + outStr + "'"
	}

	return outStr
}

func ToBackslash(path string) string {
	return strings.Replace(path, "/", "\\", -1)
}

func ToSlash(path string) string {
	return strings.Replace(path, "\\", "/", -1)
}
