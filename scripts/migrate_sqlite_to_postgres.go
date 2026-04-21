package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type targetColumn struct {
	Name     string
	DataType string
	UDTName  string
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: go run ./scripts/migrate_sqlite_to_postgres.go --sqlite ./data-prod/one-api.db --postgres \"postgresql://user:pass@127.0.0.1:5432/new-api?sslmode=disable\"\n")
	flag.PrintDefaults()
}

func main() {
	var (
		sqlitePath   string
		postgresDSN  string
		skipTruncate bool
	)

	flag.StringVar(&sqlitePath, "sqlite", "./data-prod/one-api.db", "path to the source SQLite database")
	flag.StringVar(&postgresDSN, "postgres", "", "PostgreSQL DSN")
	flag.BoolVar(&skipTruncate, "skip-truncate", false, "do not truncate PostgreSQL tables before import")
	flag.Usage = usage
	flag.Parse()

	if postgresDSN == "" {
		usage()
		os.Exit(1)
	}

	sqliteDB, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		fatalf("open sqlite failed: %v", err)
	}
	defer sqliteDB.Close()

	postgresDB, err := sql.Open("pgx", postgresDSN)
	if err != nil {
		fatalf("open postgres failed: %v", err)
	}
	defer postgresDB.Close()

	if err := sqliteDB.Ping(); err != nil {
		fatalf("ping sqlite failed: %v", err)
	}
	if err := postgresDB.Ping(); err != nil {
		fatalf("ping postgres failed: %v", err)
	}

	if err := ensurePostgresSchemaReady(postgresDB); err != nil {
		fatalf("%v", err)
	}

	tables, err := listSQLiteTables(sqliteDB)
	if err != nil {
		fatalf("list sqlite tables failed: %v", err)
	}

	tx, err := postgresDB.Begin()
	if err != nil {
		fatalf("begin postgres transaction failed: %v", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.Exec(`SET lock_timeout = '0'`); err != nil {
		fatalf("set postgres lock_timeout failed: %v", err)
	}
	if _, err := tx.Exec(`SET statement_timeout = '0'`); err != nil {
		fatalf("set postgres statement_timeout failed: %v", err)
	}
	if _, err := tx.Exec(`SET session_replication_role = replica`); err != nil {
		fmt.Printf("warning: failed to disable foreign key triggers automatically: %v\n", err)
	}
	defer func() {
		_, _ = tx.Exec(`SET session_replication_role = DEFAULT`)
	}()

	if !skipTruncate {
		if err := truncateTargetTables(tx, tables); err != nil {
			fatalf("truncate postgres tables failed: %v", err)
		}
	}

	totalRows := 0
	for _, table := range tables {
		rowsCopied, err := migrateTable(sqliteDB, tx, table)
		if err != nil {
			fatalf("migrate table %s failed: %v", table, err)
		}
		totalRows += rowsCopied
		if rowsCopied > 0 {
			fmt.Printf("migrated %-32s %d rows\n", table, rowsCopied)
		} else {
			fmt.Printf("migrated %-32s %d rows\n", table, 0)
		}
	}

	if err := tx.Commit(); err != nil {
		fatalf("commit postgres transaction failed: %v", err)
	}

	fmt.Printf("migration finished successfully, total rows copied: %d\n", totalRows)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func ensurePostgresSchemaReady(db *sql.DB) error {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = current_schema()
	`).Scan(&count)
	if err != nil {
		return fmt.Errorf("check postgres schema failed: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("postgres schema is empty, please start new-api with PostgreSQL once so it can auto-migrate tables before importing data")
	}
	return nil
}

func listSQLiteTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT name
		FROM sqlite_master
		WHERE type = 'table'
		  AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func truncateTargetTables(tx *sql.Tx, tables []string) error {
	if len(tables) == 0 {
		return nil
	}

	quoted := make([]string, 0, len(tables))
	for _, table := range tables {
		quoted = append(quoted, quoteIdent(table))
	}

	stmt := "TRUNCATE TABLE " + strings.Join(quoted, ", ") + " RESTART IDENTITY CASCADE"
	_, err := tx.Exec(stmt)
	return err
}

func migrateTable(sqliteDB *sql.DB, tx *sql.Tx, table string) (int, error) {
	sqliteColumns, err := getSQLiteColumns(sqliteDB, table)
	if err != nil {
		return 0, err
	}
	targetColumns, err := getPostgresColumns(tx, table)
	if err != nil {
		return 0, err
	}

	filteredColumns := make([]string, 0, len(sqliteColumns))
	for _, col := range sqliteColumns {
		if _, ok := targetColumns[strings.ToLower(col)]; ok {
			filteredColumns = append(filteredColumns, col)
		}
	}

	if len(filteredColumns) == 0 {
		return 0, nil
	}

	selectQuery := fmt.Sprintf(
		"SELECT %s FROM %s",
		joinQuotedColumns(filteredColumns),
		quoteIdent(table),
	)
	rows, err := sqliteDB.Query(selectQuery)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	insertQuery := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		quoteIdent(table),
		joinQuotedColumns(filteredColumns),
		joinPlaceholders(len(filteredColumns)),
	)

	stmt, err := tx.Prepare(insertQuery)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	rowCount := 0
	for rows.Next() {
		rawValues := make([]any, len(filteredColumns))
		scanTargets := make([]any, len(filteredColumns))
		for i := range rawValues {
			scanTargets[i] = &rawValues[i]
		}
		if err := rows.Scan(scanTargets...); err != nil {
			return rowCount, err
		}

		args := make([]any, len(filteredColumns))
		for i, col := range filteredColumns {
			args[i], err = normalizeValue(rawValues[i], targetColumns[strings.ToLower(col)])
			if err != nil {
				return rowCount, fmt.Errorf("normalize %s.%s failed: %w", table, col, err)
			}
		}

		if _, err := stmt.Exec(args...); err != nil {
			return rowCount, fmt.Errorf("insert into %s failed: %w", table, err)
		}
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return rowCount, err
	}

	if hasColumn(filteredColumns, "id") {
		if err := resetSequence(tx, table); err != nil {
			return rowCount, err
		}
	}

	return rowCount, nil
}

func getSQLiteColumns(db *sql.DB, table string) ([]string, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", quoteIdent(table))
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var (
			cid        int
			name       string
			colType    string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultVal, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	return columns, rows.Err()
}

func getPostgresColumns(tx *sql.Tx, table string) (map[string]targetColumn, error) {
	rows, err := tx.Query(`
		SELECT column_name, data_type, udt_name
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = $1
		ORDER BY ordinal_position
	`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[string]targetColumn)
	for rows.Next() {
		var col targetColumn
		if err := rows.Scan(&col.Name, &col.DataType, &col.UDTName); err != nil {
			return nil, err
		}
		columns[strings.ToLower(col.Name)] = col
	}
	return columns, rows.Err()
}

func normalizeValue(value any, column targetColumn) (any, error) {
	if value == nil {
		return nil, nil
	}

	switch {
	case isBoolType(column):
		return toBool(value)
	case isIntegerType(column):
		return toInt64(value)
	case isFloatType(column):
		return toFloat64(value)
	case isTextLikeType(column):
		return toStringValue(value), nil
	case isTimeLikeType(column):
		if t, ok := value.(time.Time); ok {
			return t, nil
		}
		return toStringValue(value), nil
	default:
		switch v := value.(type) {
		case []byte:
			return string(v), nil
		default:
			return v, nil
		}
	}
}

func isBoolType(column targetColumn) bool {
	return column.DataType == "boolean" || column.UDTName == "bool"
}

func isIntegerType(column targetColumn) bool {
	switch column.DataType {
	case "smallint", "integer", "bigint":
		return true
	}
	switch column.UDTName {
	case "int2", "int4", "int8":
		return true
	}
	return false
}

func isFloatType(column targetColumn) bool {
	switch column.DataType {
	case "numeric", "decimal", "real", "double precision":
		return true
	}
	switch column.UDTName {
	case "numeric", "float4", "float8":
		return true
	}
	return false
}

func isTextLikeType(column targetColumn) bool {
	switch column.DataType {
	case "text", "character varying", "character", "json", "jsonb":
		return true
	}
	switch column.UDTName {
	case "varchar", "text", "bpchar", "json", "jsonb":
		return true
	}
	return false
}

func isTimeLikeType(column targetColumn) bool {
	return strings.Contains(column.DataType, "timestamp") ||
		strings.Contains(column.DataType, "date") ||
		strings.Contains(column.DataType, "time")
}

func toBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case int64:
		return v != 0, nil
	case int32:
		return v != 0, nil
	case int:
		return v != 0, nil
	case float64:
		return v != 0, nil
	case []byte:
		return parseBoolString(string(v))
	case string:
		return parseBoolString(v)
	default:
		return false, fmt.Errorf("unsupported bool source type %T", value)
	}
}

func parseBoolString(raw string) (bool, error) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "1", "t", "true", "y", "yes":
		return true, nil
	case "0", "f", "false", "n", "no", "":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool value %q", raw)
	}
}

func toInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int32:
		return int64(v), nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	case []byte:
		return strconv.ParseInt(strings.TrimSpace(string(v)), 10, 64)
	case string:
		return strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	default:
		return 0, fmt.Errorf("unsupported integer source type %T", value)
	}
}

func toFloat64(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	case []byte:
		return strconv.ParseFloat(strings.TrimSpace(string(v)), 64)
	case string:
		return strconv.ParseFloat(strings.TrimSpace(v), 64)
	default:
		return 0, fmt.Errorf("unsupported float source type %T", value)
	}
}

func toStringValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.RFC3339Nano)
	default:
		return fmt.Sprint(v)
	}
}

func resetSequence(tx *sql.Tx, table string) error {
	var seqName sql.NullString
	if err := tx.QueryRow(`SELECT pg_get_serial_sequence($1, 'id')`, table).Scan(&seqName); err != nil {
		return err
	}
	if !seqName.Valid || seqName.String == "" {
		return nil
	}

	sequenceLiteral := strings.ReplaceAll(seqName.String, `'`, `''`)
	stmt := fmt.Sprintf(
		"SELECT setval('%s', COALESCE((SELECT MAX(%s) FROM %s), 1), (SELECT COUNT(*) > 0 FROM %s))",
		sequenceLiteral,
		quoteIdent("id"),
		quoteIdent(table),
		quoteIdent(table),
	)
	_, err := tx.Exec(stmt)
	return err
}

func joinQuotedColumns(columns []string) string {
	quoted := make([]string, 0, len(columns))
	for _, col := range columns {
		quoted = append(quoted, quoteIdent(col))
	}
	return strings.Join(quoted, ", ")
}

func joinPlaceholders(count int) string {
	parts := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		parts = append(parts, fmt.Sprintf("$%d", i))
	}
	return strings.Join(parts, ", ")
}

func hasColumn(columns []string, target string) bool {
	target = strings.ToLower(target)
	for _, col := range columns {
		if strings.ToLower(col) == target {
			return true
		}
	}
	return false
}
