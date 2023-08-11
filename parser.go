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
	At this point we use a hand-crafted recursive descent parser with two-token look-ahead.
	The result is an LL(k) model (where k = 2). https://en.wikipedia.org/wiki/LL_parser
	Each function works from a named state in the EBNF grammar (see docs/grammar.txt)
*/

type Parser struct {
	query       string        // Original query string, for error reporting and tracing
	tokens      []lexer_token // Token slice from the lexer
	num_tokens  int           // Number of tokens in the statement
	token_index int           // token index of the parser, during processing

	fields        []string // List of fields to return from query
	field_aliases []string // List of field aliases to return from query
	find_flags    byte     // ALL fields

	time_from int64 // Earliest time we want
	time_to   int64 // Latest time we want

	or_list []*or_item // base of item slice
}

const (
	find_flags_all = 0b_00000001
)

type item struct { // item leaves
	lexer_sym int
	lexer_tag *string
	lexer_val *string
}

type or_item struct { // OR items
	this     item
	left     item
	right    item
	and_list []*and_item
}

type and_item struct { // AND items (within OR)
	this  item
	left  item
	right item
}

const ( // We use the int64 unix epoch: nanoseconds since 1 Jan 1970
	temp_second    = 1000 * 1000 * 1000
	temp_minute    = temp_second * 60
	temp_hour      = temp_minute * 60
	temp_day       = temp_hour * 24
	temp_week      = temp_day * 7
	temp_fortnight = temp_day * 14
	temp_month     = temp_day * 30  // rough approximation is close enough
	temp_quarter   = temp_day * 90  // also approx
	temp_year      = temp_day * 365 // approx, since we don't care for leap year hopping
	temp_century   = temp_year * 100
)

func CurrentFunctionName() string {
	pc, _, _, _ := runtime.Caller(1)
	currentFunction := runtime.FuncForPC(pc).Name()
	return currentFunction
}

func (p *Parser) do_val_expr(newitem *item) error {
	(*newitem).lexer_sym = p.tokens[p.token_index].token
	(*newitem).lexer_tag = &(p.tokens[p.token_index].tag)
	(*newitem).lexer_val = &(p.tokens[p.token_index].val)

	return nil
}

func (p *Parser) do_and_cond() error {
	var new_and_item and_item

	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	or_ofs := len(p.or_list) - 1
	if p.or_list[or_ofs].and_list != nil {
		p.or_list[or_ofs].and_list = append(p.or_list[or_ofs].and_list, &and_item{})
	} else {
		p.or_list[or_ofs].and_list = make([]*and_item, 1, 10)
	}

	if err := p.do_val_expr(&new_and_item.left); err != nil {
		return err
	}
	p.token_index++

	if p.token_index+2 >= p.num_tokens {
		return fmt.Errorf("MATCHING statement cut short at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
	}

	switch p.tokens[p.token_index].token {
	case sym_equal:
		break
		// others to follow, will also change errormsg below
	default:
		return fmt.Errorf("expected equal (=) sign at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
	}

	p.do_val_expr(&new_and_item.this)
	p.token_index++ // Skip past comparison keyword/token

	if err := p.do_val_expr(&new_and_item.right); err != nil {
		return err
	}
	p.token_index++

	// put the and_item in the or_list
	p.or_list[or_ofs].and_list[len(p.or_list[or_ofs].and_list)-1] = &new_and_item

	return nil
}

// only do "=" and "AND" for now, whole matching-cond functionality later
func (p *Parser) do_or_cond() error {
	var new_or_item or_item

	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	if p.or_list != nil {
		p.or_list = append(p.or_list, &or_item{})
	} else {
		p.or_list = make([]*or_item, 1, 10)
	}

	if err := p.do_val_expr(&new_or_item.left); err != nil {
		return err
	}
	p.token_index++

	if p.token_index+2 >= p.num_tokens {
		return fmt.Errorf("MATCHING statement cut short at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
	}

	switch p.tokens[p.token_index].token {
	case sym_equal:
		break
		// others to follow, will also change errormsg below
	default:
		return fmt.Errorf("expected equal (=) sign at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
	}

	p.do_val_expr(&new_or_item.this)
	p.token_index++ // Skip past comparison keyword/token

	if err := p.do_val_expr(&new_or_item.right); err != nil {
		return err
	}
	p.token_index++

	// put the item in the or_list
	p.or_list[len(p.or_list)-1] = &new_or_item

	// Do we have any (more) AND clauses?
	// look-ahead(1), kinda
	for p.tokens[p.token_index].token == sym_and {
		p.token_index++

		if err := p.do_and_cond(); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) do_matching_cond() error {
	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	// First item in MATCHING clause is regarded as an OR, inside the parser structure
	if err := p.do_or_cond(); err != nil {
		return err
	}

	// Do we have any (more) OR clauses?
	// look-ahead(1), kinda
	for p.tokens[p.token_index].token == sym_or {
		p.token_index++

		if err := p.do_or_cond(); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) do_int_literal(int_literal *int) error {
	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	if i, err := strconv.Atoi(p.tokens[p.token_index].val); err != nil {
		return fmt.Errorf("not an integer literal at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
	} else {
		*int_literal = int(i)
	}

	return nil
}

// Find previous specified weekday, or the one before that
func prev_weekday(curDateTime time.Time, weekday time.Weekday, times int) time.Time {
	curDateTime = curDateTime.AddDate(0, 0, -int(curDateTime.Weekday()-weekday+7)%7)
	if times > 1 {
		curDateTime = curDateTime.AddDate(0, 0, -7)
	}

	curDateTime = curDateTime.Truncate(24 * time.Hour)

	return curDateTime
}

// Find previous specified month, or the one before that
func prev_month(curDateTime time.Time, month time.Month, times int) time.Time {
	curYear := curDateTime.Year()
	curMonth := curDateTime.Month()

	// are we prior or in the desired month this year? Then we need to step back an extra year.
	if curMonth <= month {
		times++
	}

	// Assemble datetime
	curDateTime = time.Date(int(curYear), month, 1, 0, 0, 0, 0, time.UTC) // truncated to midnight
	curDateTime = curDateTime.AddDate(-(times - 1), 0, 0)                 // hop back required # of years

	return curDateTime
}

func (p *Parser) do_reltime_ref(clock_ref *int64, int_literal int, end bool) error {
	var times int
	var tok int

	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	curDateTime := time.Now()

	// syntactically, these bits should be handled in do_temp_ref
	if (p.token_index+1) < p.num_tokens &&
		p.tokens[p.token_index].token == sym_last {
		// LAST <reltime-ref>
		tok = p.tokens[p.token_index+1].token
		times = 1
		p.token_index += 2 // skip past this whole clause, we have the necessary info in other vars
	} else if (p.token_index+2) < p.num_tokens && // look-ahead x2
		p.tokens[p.token_index+1].token == sym_before &&
		p.tokens[p.token_index+2].token == sym_last {
		// <reltime-ref> BEFORE LAST
		tok = p.tokens[p.token_index].token
		times = 2
		p.token_index += 3 // skip past this whole clause, we have the necessary info in other vars
	} else if (p.token_index+1) < p.num_tokens && // look-ahead
		p.tokens[p.token_index+1].token == sym_ago {
		// <int-literal> <reltime-ref> AGO
		// <int-literal> already parsed by caller do_temp_ref()
		times = int_literal
		tok = p.tokens[p.token_index].token
		p.token_index += 2 // skip past this whole clause, we have the necessary info in other vars
	}

	if end {
		// TODO: Need to improve on this logic - it's more complex and needs to be, per temporal range
		_ = times
		//times-- // Not perfect, but it's close enough. We're looking backwards, so - instead of +.
	}

	switch tok {
	//
	// relative clock refs (LAST HOUR, HOUR BEFORE LAST, 2 HOURS AGO)
	case sym_second:
		curDateTime = curDateTime.Add(-time.Duration(times))
	case sym_minute:
		curDateTime = curDateTime.Add(-time.Duration(60 * times))
		curDateTime = curDateTime.Truncate(time.Minute) // Truncate back to minutes
	case sym_hour:
		curDateTime = curDateTime.Add(-time.Duration(3600 * times))
		curDateTime = curDateTime.Truncate(time.Hour) // Truncate back to hours
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
		curDateTime = curDateTime.Truncate(24 * time.Hour)
	case sym_week:
		curDateTime = curDateTime.AddDate(0, 0, -7*int(times))
		curDateTime = curDateTime.Truncate(24 * time.Hour)
	case sym_fortnight:
		curDateTime = curDateTime.AddDate(0, 0, -14*int(times))
		curDateTime = curDateTime.Truncate(24 * time.Hour)
	case sym_month:
		curDateTime = curDateTime.AddDate(0, -int(times), 0)
		curDateTime = curDateTime.Truncate(24 * time.Hour)
	case sym_quarter: // We take a quarter to be just 3 months anywhere within the year
		curDateTime = curDateTime.AddDate(0, -3*int(times), 0)
		curDateTime = curDateTime.Truncate(24 * time.Hour)
	case sym_year:
		curDateTime = curDateTime.AddDate(-int(times), 0, 0)
		curDateTime = curDateTime.Truncate(24 * time.Hour)
	case sym_century:
		curDateTime = curDateTime.AddDate(-100*int(times), 0, 0)
		curDateTime = curDateTime.Truncate(24 * time.Hour)

	default:
		if int_literal == 0 {
			return fmt.Errorf("unexpected symbol at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
		}
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

	clock_ref = time.Now().UTC().UnixNano()

	switch p.tokens[p.token_index].token {
	case sym_day:
		// DAY BEFORE YESTERDAY
		if (p.token_index+2) < p.num_tokens &&
			p.tokens[p.token_index+1].token == sym_before &&
			p.tokens[p.token_index+2].token == sym_yesterday {
			clock_ref -= 2 * temp_day
			clock_ref -= clock_ref % temp_day // round back to day
			if end {
				clock_ref += temp_day - temp_second
			}
			p.token_index += 3
		} else {
			return fmt.Errorf("BEFORE YESTERDAY missing at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
		}
	case sym_yesterday:
		// YESTERDAY
		clock_ref -= temp_day
		clock_ref -= clock_ref % temp_day // round back to day
		if end {
			clock_ref += temp_day - temp_second
		}
		p.token_index++
	case sym_last:
		if error := p.do_reltime_ref(&clock_ref, int_literal, end); error != nil {
			return error
		}
	case sym_none:
		if p.tokens[p.token_index].tag == "int" {
			if error := p.do_int_literal(&int_literal); error != nil {
				return error
			}
			p.token_index++

			if error := p.do_reltime_ref(&clock_ref, int_literal, end); error != nil {
				return error
			}
		} else {
			if tt, err := time.Parse(time.DateTime, p.tokens[p.token_index].val); err == nil {
				// Could be an ISO-8601 / RFC-3339 datetime (without timezone)
				// See https://www.iso.org/iso-8601-date-and-time-format.html
				// and https://www.rfc-editor.org/rfc/rfc3339
				// TODO: test fail BETWEEN '2020-05-04' AND '2022-10-09' ends up BETWEEN 2020-05-04 10:00:00 AND 2022-10-09 10:00:00
				clock_ref = tt.UTC().UnixNano()
			} else if tt, err := time.Parse(time.DateOnly, p.tokens[p.token_index].val); err == nil {
				clock_ref = tt.UTC().UnixNano()
			} else if tt, err := time.Parse(time.TimeOnly, p.tokens[p.token_index].val); err == nil {
				clock_ref = tt.UTC().UnixNano()
			} else { // Something invalid/unknown
				return fmt.Errorf("invalid temporal reference at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
			}
			p.token_index++
		}
	default:
		// Syntactically, "... BEFORE LAST" and "... AGO" should be handled here, not in do_reltime_ref()
		if error := p.do_reltime_ref(&clock_ref, int_literal, end); error != nil {
			return error
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
		if error := p.do_temp_since(); error != nil {
			return error
		}
	case sym_between:
		p.token_index++ // skip past BETWEEN keyword
		if error := p.do_temp_between(); error != nil {
			return error
		}
	default:
		// shouldn't happen, caller do_syntax() has already picked
	}

	if p.time_from > p.time_to { // is the end time before the start time?
		p.time_from, p.time_to = p.time_to, p.time_from // swap start and end time
	}

	fmt.Fprintf(os.Stderr, "... BETWEEN %s AND %s\n", // DEBUG
		time.Unix(0, p.time_from).UTC().Format(time.DateTime), // DEBUG
		time.Unix(0, p.time_to).UTC().Format(time.DateTime))   // DEBUG

	return nil
}

func (p *Parser) do_derived_field() error {
	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

	switch p.tokens[p.token_index].tag {
	case "int":
		// TODO: not yet implemented
		break
	case "float":
		// TODO: not yet implemented
		break
	case "string":
		// TODO: not yet implemented
		break
	case "ident":
		// TODO: only implemented straight retrieval of field, with optional alias (<as-clause>)
		if p.fields == nil {
			p.fields = make([]string, 0, 100)
		}
		field := p.tokens[p.token_index].val
		p.fields = append(p.fields, field)

		if p.field_aliases == nil {
			p.field_aliases = make([]string, 0, 100)
		}
		if p.token_index+2 < p.num_tokens && p.tokens[p.token_index+1].token == sym_as { // field alias?
			p.field_aliases = append(p.field_aliases, p.tokens[p.token_index].val)
			p.token_index += 3
		} else { // no field alias
			p.field_aliases = append(p.field_aliases, field) // use main field name
			p.token_index++
		}
	default:
		return fmt.Errorf("unexpected clause in <derived-key> at '%s'", p.query[p.tokens[p.token_index].stmt_pos:])
	}

	/*
		// look-ahead (1)
		if (p.token_index+1) < p.num_tokens && p.tokens[p.token_index+1].token == sym_comma {
			p.token_index++ // skip past comma
		}
	*/

	return nil
}

func (p *Parser) do_stmt_sublist() error {
	var sublist int

	fmt.Fprintf(os.Stderr, "%s(): %v\n", CurrentFunctionName(), p.tokens[p.token_index])

exitloop:
	for p.token_index < p.num_tokens {
		switch p.tokens[p.token_index].token {
		case sym_comma:
			// comma before first <stmt-sublist>, two adjacent, or after last (using look-ahead)
			if sublist < 1 || (p.token_index+1 < p.num_tokens && p.tokens[p.token_index+1].token != sym_none) {
				return fmt.Errorf("expected <stmt-sublist> at '%s'", p.query[p.tokens[p.token_index+1].stmt_pos:])
			}
			p.token_index++
		case sym_matching:
			break exitloop // let caller deal with this
		case sym_since:
			break exitloop // let caller deal with this
		case sym_between:
			break exitloop // let caller deal with this
		case sym_none:
			sublist++
			if error := p.do_derived_field(); error != nil {
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

	fmt.Fprintf(os.Stderr, "Fields=%v\nAliases=%v\n", p.fields, p.field_aliases) // DEBUG

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
		//return fmt.Errorf("sub-commands not yet implemented: %v", cmd2)
	}

	// DEBUG
	fmt.Fprintf(os.Stderr, "Parsed OR structure:\n")
	for i := 0; i < len(p.or_list); i++ {
		fmt.Fprintf(os.Stderr, "OR %s %s %s", *p.or_list[i].left.lexer_val, *p.or_list[i].this.lexer_tag, *p.or_list[i].right.lexer_val)
		for j := 0; p.or_list != nil && j < len(p.or_list[i].and_list); j++ {
			//fmt.Fprintf(os.Stderr, " AND %v", p.or_list[i].and_list[j])
			fmt.Fprintf(os.Stderr, " AND %s %s %s", *p.or_list[i].and_list[j].left.lexer_val, *p.or_list[i].and_list[j].this.lexer_tag, *p.or_list[i].and_list[j].right.lexer_val)
		}
		fmt.Fprintln(os.Stderr)
	}
	fmt.Fprintln(os.Stderr)
	// DEBUG

	return nil // Parsing completed successfully
}

// EOF
