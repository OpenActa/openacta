// OpenActa - Parser
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
	"runtime"
	"strconv"
	"time"
)

/*
	At this point we use a hand-crafted recursive descent parser with single look-ahead.
	The result is an LL(k) model (where k = 1). https://en.wikipedia.org/wiki/LL_parser
	Each function works from a named state in the EBNF grammar (see docs/grammar.txt)
*/

type Parser struct {
	query       string        // Original query string, for error reporting and tracing
	tokens      []lexer_token // Token slice from the lexer
	num_tokens  int           // Number of tokens in the statement
	token_index int           // token index of the parser, during processing
	fields      []string      // List of fields to return from query
	find_flags  byte          // ALL
	time_from   int64         // Earliest time we want
	time_to     int64         // Latest time we want
}

const (
	find_flags_all = 0b_00000001
)

type item struct {
	lexer_sym int
	tag       string
	left      *item
	right     *item
}

const ( // We use the int64 unix epoch: nanoseconds since 1 Jan 1970
	temp_second    = 1000 * 1000 * 1000
	temp_minute    = temp_second * 60
	temp_hour      = temp_minute * 60
	temp_day       = temp_hour * 24
	temp_week      = temp_day * 7
	temp_fortnight = temp_day * 14
	temp_month     = temp_day * 30 // rough approximation is close enough
	temp_quarter   = temp_day * 90 // also approx
	temp_year      = temp_day * 365
	temp_century   = temp_year * 100
)

func CurrentFunctionName() string {
	pc, _, _, _ := runtime.Caller(1)
	currentFunction := runtime.FuncForPC(pc).Name()
	return currentFunction
}

func (p *Parser) do_matching_cond() error {
	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	// TODO

	return nil
}

func (p *Parser) do_int_literal(int_literal *int) error {
	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	if i, err := strconv.Atoi(p.tokens[p.token_index].val); err == nil {
		return fmt.Errorf("not an integer literal at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
	} else {
		*int_literal = int(i)
	}

	return nil
}

func prev_weekday(curDateTime time.Time, weekday time.Weekday, times int) time.Time {
	curDateTime = curDateTime.AddDate(0, 0, -int(curDateTime.Weekday()-weekday+7)%7)
	if times > 1 {
		curDateTime = curDateTime.AddDate(0, 0, -7)
	}
	return curDateTime
}

func prev_month(curDateTime time.Time, month time.Month, times int) time.Time {
	curDateTime = curDateTime.AddDate(0, -times, 0)
	return curDateTime
}

func (p *Parser) do_reltime_ref(clock_ref *int64, int_literal int, end bool) error {
	var times int

	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	curDateTime := time.Now()

	// syntactically, these bits should be handled in do_temp_ref
	tok := p.tokens[p.token_index].token
	if (p.token_index+1) < p.num_tokens &&
		p.tokens[p.token_index].token == sym_last {
		// LAST <reltime-ref>
		times = 1
		p.token_index += 2 // skip past this whole clause, we have the necesssary info in other vars
	} else if (p.token_index+2) < p.num_tokens &&
		p.tokens[p.token_index+1].token == sym_before &&
		p.tokens[p.token_index+2].token == sym_last {
		// <reltime-ref> BEFORE LAST
		times = 2
		// ...
		p.token_index += 3 // skip past this whole clause, we have the necesssary info in other vars
	} else {
		// n <reltime-ref> AGO
		times = int_literal
	}

	if end {
		times++ // not perfect, but it's close enough
	}

	switch tok {
	//
	// relative clock refs (LAST HOUR, HOUR BEFORE LAST, 2 HOURS AGO)
	case sym_second:
		curDateTime = curDateTime.Add(time.Duration(times))
	case sym_minute:
		curDateTime = curDateTime.Add(time.Duration(60 * times))
	case sym_hour:
		curDateTime = curDateTime.Add(time.Duration(3600 * times))
		//
		// relative weekday refs (LAST SUNDAY, SUNDAY BEFORE LAST, 2 SUNDAYS AGO), a bit more complicated
	case sym_monday:
		curDateTime = prev_weekday(curDateTime, time.Monday, times)
	case sym_tuesday:
		curDateTime = prev_weekday(curDateTime, time.Tuesday, times)
	case sym_wednesday:
		curDateTime = prev_weekday(curDateTime, time.Wednesday, times)
	case sym_thursday:
		curDateTime = prev_weekday(curDateTime, time.Thursday, times)
	case sym_friday:
		curDateTime = prev_weekday(curDateTime, time.Friday, times)
	case sym_saturday:
		curDateTime = prev_weekday(curDateTime, time.Saturday, times)
	case sym_sunday:
		curDateTime = prev_weekday(curDateTime, time.Sunday, times)
		//
		// relative month refs (LAST MAY, MAY BEFORE LAST, 2 MAYS AGO) - that last one is a bit quirky
	case sym_january:
		curDateTime = prev_month(curDateTime, 1, times)
	case sym_february:
		curDateTime = prev_month(curDateTime, 2, times)
	case sym_march:
		curDateTime = prev_month(curDateTime, 3, times)
	case sym_april:
		curDateTime = prev_month(curDateTime, 4, times)
	case sym_may:
		curDateTime = prev_month(curDateTime, 5, times)
	case sym_june:
		curDateTime = prev_month(curDateTime, 6, times)
	case sym_july:
		curDateTime = prev_month(curDateTime, 7, times)
	case sym_august:
		curDateTime = prev_month(curDateTime, 8, times)
	case sym_september:
		curDateTime = prev_month(curDateTime, 9, times)
	case sym_october:
		curDateTime = prev_month(curDateTime, 10, times)
	case sym_november:
		curDateTime = prev_month(curDateTime, 11, times)
	case sym_december:
		curDateTime = prev_month(curDateTime, 12, times)
		//
		// relative calendar refs
	case sym_day:
		curDateTime = curDateTime.AddDate(0, 0, -int(times))
	case sym_week:
		curDateTime = curDateTime.AddDate(0, 0, -7*int(times))
	case sym_fortnight:
		curDateTime = curDateTime.AddDate(0, 0, -14*int(times))
	case sym_month:
		curDateTime = curDateTime.AddDate(0, -int(times), 0)
	case sym_quarter:
		curDateTime = curDateTime.AddDate(0, -3*int(times), 0)
	case sym_year:
		curDateTime = curDateTime.AddDate(-int(times), 0, 0)
	case sym_century:
		curDateTime = curDateTime.AddDate(-100*int(times), 0, 0)
	}

	*clock_ref = curDateTime.UnixNano()

	return nil
}

// fills t with the temporal reference
// if end=true, adjust time to end of referred range
func (p *Parser) do_temp_ref(t *int64, end bool) error {
	var clock_ref int64
	var int_literal int

	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	clock_ref = time.Now().UnixNano()

	switch p.tokens[p.token_index].tag {
	case "int":
		if error := p.do_int_literal(&int_literal); error != nil {
			return error
		}
		p.token_index++

		if error := p.do_reltime_ref(&clock_ref, int_literal, end); error != nil {
			return error
		}

	default:
		switch p.tokens[p.token_index].token {
		case sym_forever:
			clock_ref = 0
		case sym_day:
			// DAY BEFORE YESTERDAY
			if (p.token_index+2) < p.num_tokens &&
				p.tokens[p.token_index+1].token == sym_before &&
				p.tokens[p.token_index+2].token == sym_yesterday {
				clock_ref -= 2*temp_day - (clock_ref % temp_day)
				if end {
					clock_ref += temp_day
				}
			} else {
				return fmt.Errorf("BEFORE YESTERDAY missing at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
			}
		case sym_yesterday:
			// YESTERDAY
			if (p.token_index+1) < p.num_tokens &&
				p.tokens[p.token_index+1].token == sym_yesterday {
				clock_ref -= temp_day - (clock_ref % temp_day)
				if end {
					clock_ref += temp_day
				}
			} else {
				return fmt.Errorf("superfluous clauses at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
			}
		case sym_last:
			if error := p.do_reltime_ref(&clock_ref, int_literal, end); error != nil {
				return error
			}
		case sym_none:
			// Could be an ISO-8601 / RFC-3339 datetime (without timezone)
			// See https://www.iso.org/iso-8601-date-and-time-format.html
			// and https://www.rfc-editor.org/rfc/rfc3339
			if tt, err := time.Parse("YYYY-MM-DD HH:MM:SS", p.tokens[p.token_index].val); err == nil {
				clock_ref = tt.UnixNano()
			} else if tt, err := time.Parse("YYYY-MM-DD HH:MM", p.tokens[p.token_index].val); err == nil {
				clock_ref = tt.UnixNano()
			} else if tt, err := time.Parse("YYYY-MM-DD", p.tokens[p.token_index].val); err == nil {
				clock_ref = tt.UnixNano()
			} else if tt, err := time.Parse("HH:MM:SS", p.tokens[p.token_index].val); err == nil {
				clock_ref = tt.UnixNano()
			} else if tt, err := time.Parse("HH:MM", p.tokens[p.token_index].val); err == nil {
				clock_ref = tt.UnixNano()
			} else {
				return fmt.Errorf("invalid temporal reference at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
			}

		default:
			// Syntactically, "... BEFORE LAST" and "... AGO" should be handled here, not in do_reltime_ref()
			if error := p.do_reltime_ref(&clock_ref, int_literal, end); error != nil {
				return error
			}
		}
	}

	*t = clock_ref
	return nil
}

func (p *Parser) do_temp_since() error {
	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	// decode desired start time
	if error := p.do_temp_ref(&p.time_from, false); error != nil {
		return error
	}

	// for "SINCE", end time is now
	p.time_to = time.Now().UnixNano()

	return nil
}

func (p *Parser) do_temp_between() error {
	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	// decode desired start time
	if error := p.do_temp_ref(&p.time_from, false); error != nil {
		return error
	}

	if p.tokens[p.token_index].token != sym_and {
		return fmt.Errorf("missing AND in <temp-between> at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
	}
	p.token_index++ // skip past AND keyword

	// decode desired end time, inclusive
	if error := p.do_temp_ref(&p.time_to, true); error != nil {
		return error
	}

	return nil
}

func (p *Parser) do_temp_cond() error {
	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	switch p.tokens[p.token_index].token {
	case sym_since:
		p.token_index++ // skip past SINCE keyword
		return p.do_temp_since()
	case sym_between:
		p.token_index++ // skip past BETWEEN keyword
		return p.do_temp_between()
	default:
		// shouldn't happen, caller do_syntax() has already picked
	}

	return nil
}

func (p *Parser) do_derived_key() error {
	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	switch p.tokens[p.token_index].tag {
	case "int":
		break
	case "float":
		break
	case "string":
		break
	case "ident":
		break
	default:
		return fmt.Errorf("unexpected clause in <derived-key> at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
	}

	// look-ahead
	if (p.token_index+1) < p.num_tokens && p.tokens[p.token_index+1].token == sym_comma {
		p.token_index++ // skip past comma
	}

	return nil
}

func (p *Parser) do_stmt_sublist() error {
	var sublist int

	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	for ; p.token_index < p.num_tokens; p.token_index++ {
		switch p.tokens[p.token_index].token {
		case sym_comma:
			// comma before first <stmt-sublist>, two adjacent, or after last (using look-ahead)
			if sublist < 1 || (p.token_index+1 < p.num_tokens && p.tokens[p.token_index+1].token != sym_none) {
				return fmt.Errorf("expected <stmt-sublist> at '%s'", p.query[p.tokens[p.token_index+1].stmt_pos:])
			}
		case sym_matching:
			return nil // let caller deal with this
		case sym_since:
			return nil // let caller deal with this
		case sym_between:
			return nil // let caller deal with this
		case sym_none:
			sublist++
			if error := p.do_derived_key(); error != nil {
				return error
			}
		default:
			if sublist < 1 {
				return fmt.Errorf("unexpected clause in <stmt-sublist> at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
			}
		}
	}

	if sublist < 1 {
		return fmt.Errorf("FIND statement cut short '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
	}

	return nil
}

func (p *Parser) do_stmt_list() error {
	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	switch p.tokens[p.token_index].token {
	case sym_all:
		p.token_index++
		p.find_flags |= find_flags_all // we are asked to return all keys
	default:
		return p.do_stmt_sublist()
	}

	return nil
}

func (p *Parser) do_stmt() error {
	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	switch p.tokens[p.token_index].token {
	case sym_find: // only statement type we have right now
		p.token_index++
		if error := p.do_stmt_list(); error != nil {
			return error
		}
	default:
		// already checked by calling function do_syntax()
	}

	return nil
}

// Top level of syntax, called by parser()
func (p *Parser) do_syntax() error {
	switch p.tokens[p.token_index].token {
	case sym_find: // only statement type we have right now
		if error := p.do_stmt(); error != nil {
			return error
		}
	default:
		return fmt.Errorf("expected statement at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
	}

	switch p.tokens[p.token_index].token {
	case sym_matching:
		p.token_index++
		if error := p.do_matching_cond(); error != nil {
			return error
		}

	default:
		// sym_matching is optional
	}

	// Temporal reference is NOT optional
	switch p.tokens[p.token_index].token {
	case sym_since:
		return p.do_temp_cond()
	case sym_between:
		return p.do_temp_cond()
	default:
		return fmt.Errorf("expected temporal clause (SINCE or BETWEEN) at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
	}

	//return nil
}

// The parser is fed a single slice of lexer tokens by application
func (p *Parser) parser() error {
	// See if there are sub-commands. If so, chop 'em so they can get processed separately.
	cmd := p.tokens
	var cmd2 []lexer_token
	for i := range cmd {
		if p.tokens[i].token == sym_pipe {
			cmd2 = cmd[i+1:]
			cmd = cmd[:i-1]
			_ = cmd
			//fmt.Fprintf(os.Stderr, "len=%d\ncmd=%v\ncmd2=%v\n", len(cmd), cmd, cmd2)	// DEBUG
			break
		}
	}

	p.num_tokens = len(p.tokens)
	p.token_index = 0 // Initialises to 0 anyway, but just to make it clear explicitly.
	error := p.do_syntax()
	if error != nil {
		return fmt.Errorf("syntax error: %s", error)
	}

	// TODO: cmd2 processing
	if len(cmd2) > 0 {
		_ = cmd2
		return fmt.Errorf("sub-commands not yet implemented: %v", cmd2)
	}

	return nil // Parsing completed successfully
}

// EOF
