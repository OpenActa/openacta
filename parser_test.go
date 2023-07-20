// OpenActa - Parser tests
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
	"os"
	"testing"
)

func TestParser(t *testing.T) {

	for i := range statements {
		tokens, error := lexer(statements[i]) // first return value is tokens array
		if error != nil {
			t.Fatalf("Lexer error: %s", error)
		}

		fmt.Fprintf(os.Stderr, "%v\n\n", tokens)

		var parser Parser
		parser.query = statements[i]
		parser.tokens = tokens
		parser.num_tokens = len(tokens)
		fmt.Fprintf(os.Stderr, "%v\n", parser)
		error = parser.parser()
		if error != nil {
			t.Fatalf("Parser error: %s", error)
		}
	}
}

// EOF
