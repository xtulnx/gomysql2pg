package cmd

import (
	"testing"
)

func TestTransformViewDef(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "nested parenthesized FROM with implicit cross join",
			input: "select b.elementname AS elementname,b.elementurl AS elementurl from ((personal_portal_element a join app_element b) join app_portal_element c on(((c.elementguid = b.RowGuid) and (a.ptrowguid = c.rowguid))))",
			want:  "select b.elementname AS elementname,b.elementurl AS elementurl FROM personal_portal_element a CROSS JOIN app_element b INNER JOIN app_portal_element c ON (((c.elementguid = b.RowGuid) and (a.ptrowguid = c.rowguid)))",
		},
		{
			name:  "simple flat join followed by WHERE",
			input: "select a.id from (t1 a join t2 b) where a.id=1",
			want:  "select a.id FROM t1 a CROSS JOIN t2 b where a.id=1",
		},
		{
			name:  "no parenthesized FROM — no change",
			input: "select a.id from t1 a, t2 b where a.id=b.id",
			want:  "select a.id from t1 a, t2 b where a.id=b.id",
		},
		{
			name:  "subquery guard — leave unchanged",
			input: "select * from (select id from t1) sub where sub.id=1",
			want:  "select * from (select id from t1) sub where sub.id=1",
		},
		{
			name:  "explicit join with ON — preserve join type",
			input: "select a.id from (t1 a left join t2 b on(a.id=b.id))",
			want:  "select a.id FROM t1 a LEFT JOIN t2 b ON (a.id=b.id)",
		},
		{
			name:  "ifnull conversion",
			input: "select ifnull(a,0) from t",
			want:  "select coalesce(a,0) from t",
		},
		{
			name:  "date_format conversion",
			input: "select date_format(d,'%Y-%m-%d') from t",
			want:  "select to_char(d,'YYYY-MM-DD') from t",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := transformViewDef(tc.input)
			if got != tc.want {
				t.Errorf("\ninput: %s\n  got: %s\n want: %s", tc.input, got, tc.want)
			}
		})
	}
}

func TestStripOuterParens(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"(abc)", "abc"},
		{"((abc))", "(abc)"},
		{"(a) join (b)", "(a) join (b)"},
		{"abc", "abc"},
		{"", ""},
	}
	for _, tc := range tests {
		got := stripOuterParens(tc.input)
		if got != tc.want {
			t.Errorf("stripOuterParens(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFullyStripOuterParens(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"(((abc)))", "abc"},
		{"(a) join (b)", "(a) join (b)"},
		{"((a join b) join c)", "(a join b) join c"},
	}
	for _, tc := range tests {
		got := fullyStripOuterParens(tc.input)
		if got != tc.want {
			t.Errorf("fullyStripOuterParens(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
