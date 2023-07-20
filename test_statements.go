// OpenActa - Test statements (used by lexer_test and parser_test)
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

var statements = []string{
	"FIND src_ip SINCE LAST HOUR | SORT src_ip",
	"FIND [dest_ip] MATCHING src_ip='192.168.0.1' AND dest_port=80 SINCE YESTERDAY | DISTINCT src_ip",
	"FIND dest_ip MATCHING src_ip='192.168.0.1' BETWEEN 3 AND 6 MONTHS AGO | SORT dest_ip",
}

// EOF
