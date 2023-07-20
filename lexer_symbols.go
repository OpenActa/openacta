// OpenActa - Lexer Symbols
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

import "regexp"

/*
We use a small hand-crafted regex-based lexer (lexer.go).
If you want to add commands or functions, this file is where you need to start.
As you see, there are multiple tables you need to add to.
Any keyword or operator in a regex needs to also be added to the symbol tables in this file
*/

type lexer_pre struct {
	regex    string
	replace  string
	compiled *regexp.Regexp
}

// Taking out line comments, block comments and distinct spacing.
// The order of these regexes can be important, so we have to use a Go slice rather than a map!
// Add new entries with care.
var lexer_pre_table = []lexer_pre{
	{regex: "//{.}\n", replace: " "},
	{regex: `/\*{.|\*}\*/`, replace: " "},
	{regex: "[\t\r\n]", replace: " "},
}

/*
The tags are mainly for debugging purposes, so we can tell which regex a match comes from.
However, they are also used by the parser.
*/

type lexer_regex struct {
	tag      string
	regex    string
	compiled *regexp.Regexp
}

// The order of these regexes is important, so we have to use a Go slice rather than a map!
// Add new entries with care.
var lexer_regex_table = []lexer_regex{
	{tag: "command", regex: `(?i)^FIND\b`},
	{tag: "cmdspec", regex: `(?i)^ALL\b`},
	{tag: "command2", regex: `(?i)^(SORT|GROUP|DISTINCT)\b`},
	{tag: "pipe", regex: `^[|]`},
	{tag: "condition", regex: `(?i)^MATCHING\b`},
	// temporal base
	{tag: "temporal", regex: `(?i)^(SINCE|BETWEEN)\b`},
	// temporal scope
	{tag: "relative", regex: `(?i)^(YESTERDAY|BEFORE|LAST|PREVIOUS|AGO|FOREVER)\b`},
	{tag: "clock", regex: `(?i)^(SECOND|MINUTE|HOUR)\b`},
	{tag: "clocks", regex: `(?i)^(SECONDS|MINUTES|HOURS)\b`},
	{tag: "calendar", regex: `(?i)^(DAY|WEEK|FORTNIGHT|MONTH|QUARTER|YEAR|CENTURY)\b`},
	{tag: "calendars", regex: `(?i)^(DAYS|WEEKS|FORTNIGHTS|MONTHS|QUARTERS|YEARS|CENTURIES)\b`},
	{tag: "weekday", regex: `(?i)^(MONDAY|TUESDAY|WEDNESDAY|THURSDAY|FRIDAY|SATURDAY|SUNDAY)\b`},
	{tag: "weekdays", regex: `(?i)^(MONDAYS|TUESDAYS|WEDNESDAYS|THURSDAYS|FRIDAYS|SATURDAYS|SUNDAYS)\b`},
	{tag: "mon", regex: `(?i)^(JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC)`},
	{tag: "months", regex: `(?i)^(JANUARY|FEBRUARY|MARCH|APRIL|MAY|JUNE|JULY|AUGUST|SEPTEMBER|OCTOBER|NOVEMBER|DECEMBER)`},
	{tag: "string", regex: `^('[^']*'|"[^"]*")`},                                    // strings (single or double quotes)
	{tag: "ident", regex: `^([a-zA-Z_][a-zA-Z_.@$]*)|(\[[a-zA-Z_][a-zA-Z_.@$]*)\]`}, // identifiers
	{tag: "int", regex: `^(\d+([eE]+?\d+)?)`},                                       // integers, optional E notation
	{tag: "float", regex: `^(\d*\.?\d+([eE][-+]?\d+)?)`},                            // floating point values
	// comma and parentheses
	{tag: "comma", regex: `^,`},    // comma
	{tag: "lparen", regex: `^[(]`}, // opening parenthesis
	{tag: "rparen", regex: `^[)]`}, // closing parenthesis
	// Binary operands
	{tag: "minus", regex: `^-`},           // minus/sign
	{tag: "plus", regex: `^[+]`},          // plus
	{tag: "equal", regex: `^=|==`},        // equal
	{tag: "not_equal", regex: `^(!=|<>)`}, // not equal
	{tag: "mul", regex: `^\*`},            // multiply
	{tag: "div", regex: `(?i)^(/|DIV)\b`}, // divide
	{tag: "mod", regex: `(?i)^(%|MOD)\b`}, // modulo
	{tag: "less_equal", regex: `^<=`},     // lesser or equal
	{tag: "greater_equal", regex: `^>=`},  // greater or equal
	{tag: "less", regex: `^<`},            // less
	{tag: "greater", regex: `^>`},         // greater
	// Binary operators
	{tag: "and", regex: `(?i)^AND\b`}, // AND
	{tag: "or", regex: `(?i)^OR\b`},   // OR
	// Unary operator
	{tag: "not", regex: `(?i)^(!|NOT)\b`}, // NOT
	// pattern matcher
	{tag: "like", regex: `(?i)^LIKE\b`}, // LIKE
}

// Enumeration of all symbols, order doesn't matter as long as "sym_none = iota" is first
const (
	sym_none = iota
	sym_find
	sym_sort
	sym_group
	sym_distinct
	sym_all
	sym_pipe
	sym_matching
	sym_since
	sym_between
	sym_yesterday
	sym_before
	sym_last
	sym_previous
	sym_ago
	sym_forever
	sym_second
	sym_minute
	sym_hour
	sym_day
	sym_week
	sym_fortnight
	sym_month
	sym_quarter
	sym_year
	sym_century
	sym_monday
	sym_tuesday
	sym_wednesday
	sym_thursday
	sym_friday
	sym_saturday
	sym_sunday
	sym_january
	sym_february
	sym_march
	sym_april
	sym_may
	sym_june
	sym_july
	sym_august
	sym_september
	sym_october
	sym_november
	sym_december
	sym_comma
	sym_lparen
	sym_rparen
	sym_minus
	sym_plus
	sym_equal
	sym_not_equal
	sym_mul
	sym_div
	sym_mod
	sym_less_equal
	sym_greater_equal
	sym_less
	sym_greater
	sym_and
	sym_or
	sym_not
	sym_like
)

// string -> symbol look-up, order does not matter as long as everything is in here.
var lexer_symbol_table = map[string]int{
	// Commands
	"FIND":     sym_find,
	"SORT":     sym_sort,
	"GROUP":    sym_group,
	"DISTINCT": sym_distinct,
	"ALL":      sym_all,
	"|":        sym_pipe,
	"MATCHING": sym_matching,
	// Temporals
	"SINCE": sym_since, "BETWEEN": sym_between,
	"YESTERDAY": sym_yesterday, "BEFORE": sym_before, "LAST": sym_last,
	"PREVIOUS": sym_previous, "AGO": sym_ago, "FOREVER": sym_between,
	"SECOND": sym_second, "MINUTE": sym_minute, "HOUR": sym_hour,
	"SECONDS": sym_second, "MINUTES": sym_minute, "HOURS": sym_hour,
	"DAY": sym_day, "WEEK": sym_week, "FORTNIGHT": sym_fortnight, "MONTH": sym_month,
	"DAYS": sym_day, "WEEKS": sym_week, "FORTNIGHTS": sym_fortnight, "MONTHS": sym_month,
	"QUARTER": sym_quarter, "YEAR": sym_year, "CENTURY": sym_century,
	"QUARTERS": sym_quarter, "YEARS": sym_year, "CENTURIES": sym_century,
	"MONDAY": sym_monday, "TUESDAY": sym_tuesday, "WEDNESDAY": sym_wednesday,
	"MONDAYS": sym_monday, "TUESDAYS": sym_tuesday, "WEDNESDAYS": sym_wednesday,
	"THURSDAY": sym_thursday, "FRIDAY": sym_friday,
	"SATURDAY": sym_saturday, "SUNDAY": sym_sunday,
	"THURSDAYS": sym_thursday, "FRIDAYS": sym_friday,
	"SATURDAYS": sym_saturday, "SUNDAYS": sym_sunday,
	"JAN": sym_january, "FEB": sym_february, "MAR": sym_march,
	"APR": sym_april, "MAY": sym_may, "JUN": sym_june,
	"JUL": sym_july, "AUG": sym_august, "SEP": sym_september,
	"OCT": sym_october, "NOV": sym_november, "DEC": sym_december,
	"JANUARY": sym_january, "FEBUARY": sym_february, "MARCH": sym_march,
	"APRIL": sym_april /* MAY dup */, "JUNE": sym_june,
	"JULY": sym_july, "AUGUST": sym_august, "SEPTEMBER": sym_september,
	"OCTOBER": sym_october, "NOVEMBER": sym_november, "DECEMBER": sym_december,
	// Operators
	",": sym_comma, "(": sym_lparen, ")": sym_rparen,
	"-": sym_minus, "+": sym_plus,
	"=": sym_equal, "<>": sym_not_equal, "!=": sym_not_equal,
	"*": sym_mul, "/": sym_div, "DIV": sym_div, "%": sym_mod, "MOD": sym_mod,
	"<=": sym_less_equal, ">=": sym_greater_equal, "<": sym_less, ">": sym_greater,
	"AND": sym_and, "OR": sym_or,
	"NOT": sym_not, "!": sym_not,
	"LIKE": sym_like,
	// Functions
}

// Lexer token structure, an array of these is passed to the parser
type lexer_token struct {
	tag      string // regex tag from the regex pattern array
	token    int    // token, or 0 for literals and identifiers
	val      string // value for literals and identifiers, or ""
	stmt_pos int    // position of this token in the query string
}

// EOF
