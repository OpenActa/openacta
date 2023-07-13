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
The symbol and token tables are in lexer_symbols.go
*/

// token lexer using regular expressions
func lexer(s string) ([]lexer_token, error) {
	linecomment_pattern := regexp.MustCompile("//{.}\n")
	blockcomment_pattern := regexp.MustCompile(`/\*{.|\*}\*/`)
	whitespace_pattern := regexp.MustCompile(" \t\r\n")

	// first get rid of comment fluff, and make single-spaced
	s = linecomment_pattern.ReplaceAllLiteralString(s, " ")
	s = blockcomment_pattern.ReplaceAllLiteralString(s, " ")
	s = whitespace_pattern.ReplaceAllLiteralString(s, " ")
	// Remove leading and trailing whitespaces
	s = strings.TrimSpace(s)

	// Compile our regexes
	for i := range lexer_regex_table {
		lexer_regex_table[i].compiled = regexp.MustCompile(lexer_regex_table[i].regex)
	}

	// Tokenise the statement
	var tokens []lexer_token

	// Tokenise statement (s)
	for len(s) > 0 {
		// Try match each regular expression pattern, in order
		match := false
		for i := range lexer_regex_table {
			if result := lexer_regex_table[i].compiled.FindString(s); result != "" {
				var newtoken lexer_token

				switch lexer_regex_table[i].tag {
				case "string": // remove quotes
					result = result[1 : len(result)-1]
				case "int":
				case "float":
				case "ident": // values and identifiers are not in the token table
				default: // the rest are (or should be!) in the token table
					token, exists := lexer_symbol_table[result]
					if exists {
						newtoken.token = token
					} else {
						// This can only happen if someone stuffs up in the lexer_symbols.go file
						return nil, fmt.Errorf("unknown token '%s' in lexer symbol table", result)
					}
				}

				newtoken.tag = lexer_regex_table[i].tag
				newtoken.val = result

				tokens = append(tokens, newtoken)

				s = strings.TrimSpace(lexer_regex_table[i].compiled.ReplaceAllString(s, ""))
				match = true
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
