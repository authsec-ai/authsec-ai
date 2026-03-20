package migration

import (
	"database/sql"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

type SchemaComparator struct {
	db *sql.DB
}

func NewSchemaComparator(db *sql.DB) *SchemaComparator {
	return &SchemaComparator{db: db}
}

type TableSchema struct {
	Name    string
	Columns []ColumnInfo
	Indexes []IndexInfo
}

type ColumnInfo struct {
	Name         string
	DataType     string
	IsNullable   bool
	DefaultValue *string
}

type IndexInfo struct {
	Name     string
	Columns  []string
	IsUnique bool
}

func (sc *SchemaComparator) GetTableSchema(tableName string) (*TableSchema, error) {
	schema := &TableSchema{Name: tableName}

	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = $1
		ORDER BY ordinal_position
	`

	rows, err := sc.db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var col ColumnInfo
		var isNullableStr string
		err := rows.Scan(&col.Name, &col.DataType, &isNullableStr, &col.DefaultValue)
		if err != nil {
			return nil, err
		}
		col.IsNullable = isNullableStr == "YES"
		schema.Columns = append(schema.Columns, col)
	}

	indexQuery := `
		SELECT i.relname, a.attname, ix.indisunique
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE t.relname = $1
		ORDER BY i.relname, a.attnum
	`

	indexRows, err := sc.db.Query(indexQuery, tableName)
	if err != nil {
		log.Warnf("Failed to query indexes for %s: %v", tableName, err)
	} else {
		defer indexRows.Close()

		indexMap := make(map[string]*IndexInfo)
		for indexRows.Next() {
			var indexName, columnName string
			var isUnique bool
			if err := indexRows.Scan(&indexName, &columnName, &isUnique); err != nil {
				continue
			}

			if idx, exists := indexMap[indexName]; exists {
				idx.Columns = append(idx.Columns, columnName)
			} else {
				indexMap[indexName] = &IndexInfo{
					Name:     indexName,
					Columns:  []string{columnName},
					IsUnique: isUnique,
				}
			}
		}

		for _, idx := range indexMap {
			schema.Indexes = append(schema.Indexes, *idx)
		}
	}

	return schema, nil
}

func (sc *SchemaComparator) CompareSchemas(table1, table2 string) ([]string, error) {
	schema1, err := sc.GetTableSchema(table1)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema for %s: %w", table1, err)
	}

	schema2, err := sc.GetTableSchema(table2)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema for %s: %w", table2, err)
	}

	var differences []string

	columnMap1 := make(map[string]ColumnInfo)
	for _, col := range schema1.Columns {
		columnMap1[col.Name] = col
	}

	columnMap2 := make(map[string]ColumnInfo)
	for _, col := range schema2.Columns {
		columnMap2[col.Name] = col
	}

	for name, col1 := range columnMap1 {
		if col2, exists := columnMap2[name]; !exists {
			differences = append(differences, fmt.Sprintf("Column %s exists in %s but not in %s", name, table1, table2))
		} else if col1.DataType != col2.DataType {
			differences = append(differences, fmt.Sprintf("Column %s has different type: %s vs %s", name, col1.DataType, col2.DataType))
		}
	}

	for name := range columnMap2 {
		if _, exists := columnMap1[name]; !exists {
			differences = append(differences, fmt.Sprintf("Column %s exists in %s but not in %s", name, table2, table1))
		}
	}

	return differences, nil
}

func (sc *SchemaComparator) GenerateAlterSQL(sourceTable, targetTable string) (string, error) {
	diffs, err := sc.CompareSchemas(sourceTable, targetTable)
	if err != nil {
		return "", err
	}

	if len(diffs) == 0 {
		return "-- Schemas are identical", nil
	}

	var alterStatements []string
	alterStatements = append(alterStatements, fmt.Sprintf("-- Schema differences between %s and %s:", sourceTable, targetTable))

	for _, diff := range diffs {
		alterStatements = append(alterStatements, fmt.Sprintf("-- %s", diff))
	}

	schema, err := sc.GetTableSchema(sourceTable)
	if err != nil {
		return "", err
	}

	targetSchema, err := sc.GetTableSchema(targetTable)
	if err != nil {
		return "", err
	}

	targetCols := make(map[string]bool)
	for _, col := range targetSchema.Columns {
		targetCols[col.Name] = true
	}

	alterStatements = append(alterStatements, "")
	alterStatements = append(alterStatements, fmt.Sprintf("-- Suggested ALTER statements to align %s with %s:", targetTable, sourceTable))

	for _, col := range schema.Columns {
		if !targetCols[col.Name] {
			nullable := "NOT NULL"
			if col.IsNullable {
				nullable = "NULL"
			}

			defaultClause := ""
			if col.DefaultValue != nil {
				defaultClause = fmt.Sprintf(" DEFAULT %s", *col.DefaultValue)
			}

			alterStatements = append(alterStatements,
				fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s %s%s;",
					targetTable, col.Name, col.DataType, nullable, defaultClause))
		}
	}

	return strings.Join(alterStatements, "\n"), nil
}

func (sc *SchemaComparator) DumpSchema(tableName string) (string, error) {
	schema, err := sc.GetTableSchema(tableName)
	if err != nil {
		return "", err
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Table: %s", tableName))
	lines = append(lines, "Columns:")

	for _, col := range schema.Columns {
		nullable := "NOT NULL"
		if col.IsNullable {
			nullable = "NULL"
		}

		defaultVal := "no default"
		if col.DefaultValue != nil {
			defaultVal = *col.DefaultValue
		}

		lines = append(lines, fmt.Sprintf("  - %s: %s %s (default: %s)",
			col.Name, col.DataType, nullable, defaultVal))
	}

	if len(schema.Indexes) > 0 {
		lines = append(lines, "\nIndexes:")
		for _, idx := range schema.Indexes {
			uniqueStr := ""
			if idx.IsUnique {
				uniqueStr = "UNIQUE "
			}
			lines = append(lines, fmt.Sprintf("  - %s%s on (%s)",
				uniqueStr, idx.Name, strings.Join(idx.Columns, ", ")))
		}
	}

	return strings.Join(lines, "\n"), nil
}
