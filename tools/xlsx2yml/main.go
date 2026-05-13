// Command xlsx2yml reads a db migration info xlsx and emits one yml config
// per row, matching the style of configs/01_a.yml.
//
// Usage:
//
//	go run ./tools/xlsx2yml [-f configs/db_mig_info.xlsx] [-o configs] \
//	    [--sheet NAME] [--schema-mapping] [--overwrite]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

type row struct {
	srcHost, srcPort, srcDB, srcUser, srcPwd string
	dstHost, dstPort, dstDB, dstUser, dstPwd string
	lineNo                                   int
}

var requiredCols = []string{
	"src_host", "src_port", "src_database", "src_username", "src_password",
	"dest_host", "dest_port", "dest_database", "dest_username", "dest_password",
}

func main() {
	var (
		file          string
		outDir        string
		sheet         string
		enableMapping bool
		overwrite     bool
	)
	flag.StringVar(&file, "file", "configs/db_mig_info.xlsx", "xlsx path")
	flag.StringVar(&file, "f", "configs/db_mig_info.xlsx", "xlsx path (shorthand)")
	flag.StringVar(&outDir, "out", "configs", "output dir")
	flag.StringVar(&outDir, "o", "configs", "output dir (shorthand)")
	flag.StringVar(&sheet, "sheet", "", "sheet name (default: first sheet)")
	flag.BoolVar(&enableMapping, "schema-mapping", false, "emit schemaMapping block (src_database -> dest_username)")
	flag.BoolVar(&overwrite, "overwrite", false, "overwrite existing yml files")
	flag.Parse()

	rows, err := readRows(file, sheet)
	if err != nil {
		die(err, 2)
	}
	if err := checkDup(rows); err != nil {
		die(err, 3)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		die(err, 2)
	}

	gen := 0
	for i, r := range rows {
		seq := fmt.Sprintf("%03d", i+1)
		path := filepath.Join(outDir, seq+"_"+r.srcDB+".yml")
		if !overwrite {
			if _, err := os.Stat(path); err == nil {
				fmt.Fprintf(os.Stderr, "skip existing: %s\n", path)
				continue
			}
		}
		if err := writeYML(path, r, enableMapping); err != nil {
			die(err, 2)
		}
		gen++
	}
	fmt.Printf("generated %d yml file(s) under %s\n", gen, outDir)
}

func readRows(path, sheet string) ([]row, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("xlsx not found: %s", path)
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	if sheet == "" {
		names := f.GetSheetList()
		if len(names) == 0 {
			return nil, fmt.Errorf("no sheets in %s", path)
		}
		sheet = names[0]
	}
	raw, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("read sheet %q: %w", sheet, err)
	}
	if len(raw) < 2 {
		return nil, fmt.Errorf("sheet %q has no data rows", sheet)
	}

	header := raw[0]
	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	var missing []string
	for _, c := range requiredCols {
		if _, ok := idx[c]; !ok {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required column(s) in header: %s", strings.Join(missing, ", "))
	}

	cell := func(r []string, name string) string {
		i := idx[name]
		if i >= len(r) {
			return ""
		}
		return strings.TrimRight(strings.TrimSpace(r[i]), "\r")
	}

	out := make([]row, 0, len(raw)-1)
	for n := 1; n < len(raw); n++ {
		r := raw[n]
		lineNo := n + 1 // 1-based spreadsheet row

		rr := row{
			srcHost: cell(r, "src_host"),
			srcPort: cell(r, "src_port"),
			srcDB:   cell(r, "src_database"),
			srcUser: cell(r, "src_username"),
			srcPwd:  cell(r, "src_password"),
			dstHost: cell(r, "dest_host"),
			dstPort: cell(r, "dest_port"),
			dstDB:   cell(r, "dest_database"),
			dstUser: cell(r, "dest_username"),
			dstPwd:  cell(r, "dest_password"),
			lineNo:  lineNo,
		}
		if rr.srcHost == "" && rr.srcDB == "" && rr.dstHost == "" {
			continue // fully empty row
		}
		if rr.srcHost == "" || rr.srcPort == "" || rr.srcDB == "" || rr.srcUser == "" || rr.srcPwd == "" ||
			rr.dstHost == "" || rr.dstPort == "" || rr.dstDB == "" || rr.dstUser == "" || rr.dstPwd == "" {
			fmt.Fprintf(os.Stderr, "warn: line %d incomplete, skip\n", lineNo)
			continue
		}
		if _, err := strconv.Atoi(rr.srcPort); err != nil {
			fmt.Fprintf(os.Stderr, "warn: line %d src_port %q is not a number, using as-is\n", lineNo, rr.srcPort)
		}
		if _, err := strconv.Atoi(rr.dstPort); err != nil {
			fmt.Fprintf(os.Stderr, "warn: line %d dest_port %q is not a number, using as-is\n", lineNo, rr.dstPort)
		}
		out = append(out, rr)
	}
	return out, nil
}

func checkDup(rs []row) error {
	seen := make(map[string]int, len(rs))
	var msgs []string
	for _, r := range rs {
		key := strings.Join([]string{
			r.srcHost, r.srcPort, r.srcDB, r.srcUser, r.srcPwd,
			r.dstHost, r.dstPort, r.dstDB, r.dstUser, r.dstPwd,
		}, "\x1f")
		if prev, ok := seen[key]; ok {
			msgs = append(msgs, fmt.Sprintf("duplicate row detected: line %d duplicates line %d", r.lineNo, prev))
		} else {
			seen[key] = r.lineNo
		}
	}
	if len(msgs) > 0 {
		return fmt.Errorf("%s\naborted: please fix duplicate rows", strings.Join(msgs, "\n"))
	}
	return nil
}

func writeYML(path string, r row, mapping bool) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "src:\n")
	fmt.Fprintf(f, "  host: %q\n", r.srcHost)
	fmt.Fprintf(f, "  port: %s\n", r.srcPort)
	fmt.Fprintf(f, "  database: %q\n", r.srcDB)
	fmt.Fprintf(f, "  username: %q\n", r.srcUser)
	fmt.Fprintf(f, "  password: %q\n", r.srcPwd)
	fmt.Fprintf(f, "\n")
	fmt.Fprintf(f, "dest:\n")
	fmt.Fprintf(f, "  dbType: Gauss\n")
	fmt.Fprintf(f, "  host: %s\n", r.dstHost)
	fmt.Fprintf(f, "  port: %s\n", r.dstPort)
	fmt.Fprintf(f, "  database: %s\n", r.dstDB)
	fmt.Fprintf(f, "  username: %s\n", r.dstUser)
	fmt.Fprintf(f, "  password: %s\n", r.dstPwd)
	fmt.Fprintf(f, "\n")
	fmt.Fprintf(f, "pageSize: 100000\n")
	fmt.Fprintf(f, "maxParallel: 32\n")
	fmt.Fprintf(f, "charInLength: false\n")
	fmt.Fprintf(f, "useNvarchar2: true\n")
	fmt.Fprintf(f, "Distributed: false\n")
	fmt.Fprintf(f, "tables:\n")
	fmt.Fprintf(f, "  pres_fieldinfo:\n")
	fmt.Fprintf(f, "    - select * from pres_fieldinfo\n")
	fmt.Fprintf(f, "exclude:\n")
	fmt.Fprintf(f, "  - 'xmllog_copy1'\n")
	fmt.Fprintf(f, "  - 'interfacecalllog_copy1'\n")
	fmt.Fprintf(f, "  - '*_cswysk'\n")
	if mapping {
		fmt.Fprintf(f, "schemaMapping:\n")
		fmt.Fprintf(f, "  %s: %s\n", r.srcDB, r.dstUser)
	}
	return nil
}

func die(err error, code int) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(code)
}
