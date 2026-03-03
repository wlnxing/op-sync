package openlistsync

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	dayMatchWindowMinutes = 60 * 24 * 366 * 5
)

type crontabField struct {
	any    bool
	values map[int]struct{}
}

func (f crontabField) match(v int) bool {
	if f.any {
		return true
	}
	_, ok := f.values[v]
	return ok
}

type CrontabSchedule struct {
	expr       string
	minute     crontabField
	hour       crontabField
	dayOfMonth crontabField
	month      crontabField
	dayOfWeek  crontabField
}

func (s *CrontabSchedule) Expr() string {
	if s == nil {
		return ""
	}
	return s.expr
}

// ParseCrontab 解析 5 段式 crontab: 分 时 日 月 周。
func ParseCrontab(expr string) (*CrontabSchedule, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("crontab is empty")
	}
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("crontab must have 5 fields: minute hour day month weekday")
	}

	minute, err := parseCrontabField(parts[0], 0, 59, false)
	if err != nil {
		return nil, fmt.Errorf("invalid minute field: %w", err)
	}
	hour, err := parseCrontabField(parts[1], 0, 23, false)
	if err != nil {
		return nil, fmt.Errorf("invalid hour field: %w", err)
	}
	dayOfMonth, err := parseCrontabField(parts[2], 1, 31, false)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-month field: %w", err)
	}
	month, err := parseCrontabField(parts[3], 1, 12, false)
	if err != nil {
		return nil, fmt.Errorf("invalid month field: %w", err)
	}
	dayOfWeek, err := parseCrontabField(parts[4], 0, 7, true)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-week field: %w", err)
	}

	return &CrontabSchedule{
		expr:       expr,
		minute:     minute,
		hour:       hour,
		dayOfMonth: dayOfMonth,
		month:      month,
		dayOfWeek:  dayOfWeek,
	}, nil
}

func parseCrontabField(part string, minV, maxV int, dow bool) (crontabField, error) {
	part = strings.TrimSpace(part)
	if part == "" {
		return crontabField{}, fmt.Errorf("empty field")
	}
	if part == "*" {
		return crontabField{any: true}, nil
	}

	out := crontabField{values: make(map[int]struct{})}
	items := strings.Split(part, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			return crontabField{}, fmt.Errorf("invalid list item")
		}
		if err := addFieldItem(out.values, item, minV, maxV, dow); err != nil {
			return crontabField{}, err
		}
	}
	if len(out.values) == 0 {
		return crontabField{}, fmt.Errorf("field has no values")
	}
	return out, nil
}

func addFieldItem(out map[int]struct{}, item string, minV, maxV int, dow bool) error {
	base := item
	step := 1
	if strings.Contains(item, "/") {
		parts := strings.Split(item, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid step syntax: %s", item)
		}
		base = strings.TrimSpace(parts[0])
		n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || n <= 0 {
			return fmt.Errorf("invalid step value: %s", item)
		}
		step = n
	}

	start, end, err := parseFieldRange(base, minV, maxV)
	if err != nil {
		return err
	}
	for v := start; v <= end; v++ {
		if (v-start)%step != 0 {
			continue
		}
		if err := addFieldValue(out, v, minV, maxV, dow); err != nil {
			return err
		}
	}
	return nil
}

func parseFieldRange(base string, minV, maxV int) (int, int, error) {
	base = strings.TrimSpace(base)
	if base == "*" {
		return minV, maxV, nil
	}
	if strings.Contains(base, "-") {
		rg := strings.Split(base, "-")
		if len(rg) != 2 {
			return 0, 0, fmt.Errorf("invalid range: %s", base)
		}
		start, err1 := strconv.Atoi(strings.TrimSpace(rg[0]))
		end, err2 := strconv.Atoi(strings.TrimSpace(rg[1]))
		if err1 != nil || err2 != nil {
			return 0, 0, fmt.Errorf("invalid range: %s", base)
		}
		if start > end {
			return 0, 0, fmt.Errorf("range start > end: %s", base)
		}
		return start, end, nil
	}
	v, err := strconv.Atoi(base)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid value: %s", base)
	}
	return v, v, nil
}

func addFieldValue(out map[int]struct{}, v, minV, maxV int, dow bool) error {
	if dow && v == 7 {
		v = 0
	}
	maxAllowed := maxV
	if dow {
		maxAllowed = 7
	}
	if v < minV || v > maxAllowed {
		return fmt.Errorf("value out of range: %d", v)
	}
	out[v] = struct{}{}
	return nil
}

func (s *CrontabSchedule) matches(t time.Time) bool {
	if s == nil {
		return false
	}
	if !s.month.match(int(t.Month())) || !s.hour.match(t.Hour()) || !s.minute.match(t.Minute()) {
		return false
	}

	domMatch := s.dayOfMonth.match(t.Day())
	dowMatch := s.dayOfWeek.match(int(t.Weekday()))
	switch {
	case s.dayOfMonth.any && s.dayOfWeek.any:
		return true
	case s.dayOfMonth.any:
		return dowMatch
	case s.dayOfWeek.any:
		return domMatch
	default:
		return domMatch || dowMatch
	}
}

// Next 返回 after 之后的下一次触发时间（精确到分钟）。
func (s *CrontabSchedule) Next(after time.Time) (time.Time, error) {
	if s == nil {
		return time.Time{}, fmt.Errorf("schedule is nil")
	}
	t := after.Truncate(time.Minute).Add(time.Minute)
	for i := 0; i < dayMatchWindowMinutes; i++ {
		if s.matches(t) {
			return t, nil
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, fmt.Errorf("no matching time found within 5 years for %q", s.expr)
}
