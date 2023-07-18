// OpenActa - Lexer
// Copyright (C) 2023 Arjen Lentz & Lentz Pty Ltd; All Rights Reserved
// <arjen (at) openacta (dot) dev>

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package openacta

import (
	"fmt"
	"regexp"
	"strings"
)

/*
We use a small hand-crafted regex-based lexer.
The regex, symbol and token tables are in lexer_symbols.go
*/

// The Go runtime will execute this once at startup, before calling main()
func init() {
	// Compile spacing and comments regexes
	for i := range lexer_pre_table {
		lexer_pre_table[i].compiled = regexp.MustCompile(lexer_pre_table[i].regex)
	}

	// Compile our syntax regexes
	for i := range lexer_regex_table {
		lexer_regex_table[i].compiled = regexp.MustCompile(lexer_regex_table[i].regex)
	}
}

// token lexer using regular expressions
func lexer(s string) ([]lexer_token, error) {
	// first get rid of comment fluff, and take out special spacing and CR/LF
	for i := range lexer_pre_table {
		s = lexer_pre_table[i].compiled.ReplaceAllLiteralString(s, lexer_pre_table[i].replace)
	}

	// Remove leading and trailing whitespaces
	s = strings.TrimSpace(s)

	// Tokenise the statement
	var tokens []lexer_token
	var stmt_pos int

	// Tokenise statement(s)
	for len(s) > 0 {
		// Try match each regular expression pattern, in order
		match := false
		for i := range lexer_regex_table {
			if result := lexer_regex_table[i].compiled.FindString(s); result != "" {
				var newtoken lexer_token

				switch lexer_regex_table[i].tag {
				case "string": // remove quotes
					result = result[1 : len(result)-1]
				case "ident": // values and identifiers are not in the token table
					result = strings.Trim(result, "[]") // remove brackets - would also accept [[field]] but meh
				case "int":
				case "float":
				default: // the rest are (or should be!) in the token table
					token, exists := lexer_symbol_table[result]
					if exists {
						newtoken.token = token
					} else {
						// This can only happen if someone stuffs up in the lexer_symbols.go file
						return nil, fmt.Errorf("lexer: token '%s' from regex table unknown in symbol table", result)
					}
				}

				newtoken.tag = lexer_regex_table[i].tag
				newtoken.val = result
				newtoken.stmt_pos = stmt_pos

				tokens = append(tokens, newtoken)

				s2 := lexer_regex_table[i].compiled.ReplaceAllString(s, "") // remove this token
				s2 = strings.TrimSpace(s2)                                  // remove surrounding whitespace (if applicable)
				stmt_pos += len(s) - len(s2)                                // start of next token
				s = s2

				match = true // we found a match
				break
			}
		}

		if !match {
			return nil, fmt.Errorf("unknown token or unquoted string at '%s'", s)
		}
	}

	return tokens, nil
}

// EOF
