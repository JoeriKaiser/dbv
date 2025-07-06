package database

import (
	"database/sql"
	"dbv/internal/schema"
	"dbv/pkg/config"
	"strings"
	"time"
)

type PostgreSQLExtractor struct {
    db *sql.DB
}

func (p *PostgreSQLExtractor) ExtractSchema(cfg config.SchemaConfig) (*schema.Schema, error) {
    s := &schema.Schema{
        Database:    "postgresql",
        GeneratedAt: time.Now(),
    }

    tables, err := p.extractTables(cfg)
    if err != nil {
        return nil, err
    }
    s.Tables = tables

    if cfg.IncludeViews {
        views, err := p.extractViews(cfg)
        if err != nil {
            return nil, err
        }
        s.Views = views
    }

    foreignKeys, err := p.extractForeignKeys(cfg)
    if err != nil {
        return nil, err
    }
    s.ForeignKeys = foreignKeys

    return s, nil
}

func (p *PostgreSQLExtractor) extractTables(cfg config.SchemaConfig) ([]schema.Table, error) {
    query := `
        SELECT t.table_name, t.table_type, COALESCE(obj_description(c.oid), '') as comment
        FROM information_schema.tables t
        LEFT JOIN pg_class c ON c.relname = t.table_name
        WHERE t.table_schema = 'public' AND t.table_type = 'BASE TABLE'
        ORDER BY t.table_name
    `

    rows, err := p.db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var tables []schema.Table
    for rows.Next() {
        var table schema.Table
        if err := rows.Scan(&table.Name, &table.Type, &table.Comment); err != nil {
            return nil, err
        }

        if len(cfg.IncludeTables) > 0 && !contains(cfg.IncludeTables, table.Name) {
            continue
        }
        if contains(cfg.ExcludeTables, table.Name) {
            continue
        }

        table.Schema = "public"
        
        columns, err := p.extractColumns(table.Name)
        if err != nil {
            return nil, err
        }
        table.Columns = columns

        primaryKeys, err := p.extractPrimaryKeys(table.Name)
        if err != nil {
            return nil, err
        }
        table.PrimaryKeys = primaryKeys

        tables = append(tables, table)
    }

    return tables, nil
}

func (p *PostgreSQLExtractor) extractColumns(tableName string) ([]schema.Column, error) {
    query := `
        SELECT 
            c.column_name, 
            c.data_type,
            c.character_maximum_length,
            c.numeric_precision,
            c.numeric_scale,
            c.is_nullable = 'YES' as is_nullable,
            c.column_default,
            COALESCE(col_description(pgc.oid, c.ordinal_position), '') as comment
        FROM information_schema.columns c
        LEFT JOIN pg_class pgc ON pgc.relname = c.table_name
        WHERE c.table_schema = 'public' AND c.table_name = $1
        ORDER BY c.ordinal_position
    `

    rows, err := p.db.Query(query, tableName)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var columns []schema.Column
    for rows.Next() {
        var col schema.Column
        var length, precision, scale sql.NullInt64
        var defaultValue sql.NullString

        if err := rows.Scan(
            &col.Name,
            &col.Type,
            &length,
            &precision,
            &scale,
            &col.IsNullable,
            &defaultValue,
            &col.Comment,
        ); err != nil {
            return nil, err
        }

        if length.Valid {
            l := int(length.Int64)
            col.Length = &l
        }
        if precision.Valid {
            p := int(precision.Int64)
            col.Precision = &p
        }
        if scale.Valid {
            s := int(scale.Int64)
            col.Scale = &s
        }
        if defaultValue.Valid {
            col.DefaultValue = &defaultValue.String
        }

        columns = append(columns, col)
    }

    return columns, nil
}

func (p *PostgreSQLExtractor) extractPrimaryKeys(tableName string) ([]string, error) {
    query := `
        SELECT kcu.column_name
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu 
            ON tc.constraint_name = kcu.constraint_name
        WHERE tc.table_schema = 'public' 
            AND tc.table_name = $1 
            AND tc.constraint_type = 'PRIMARY KEY'
        ORDER BY kcu.ordinal_position
    `

    rows, err := p.db.Query(query, tableName)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var primaryKeys []string
    for rows.Next() {
        var columnName string
        if err := rows.Scan(&columnName); err != nil {
            return nil, err
        }
        primaryKeys = append(primaryKeys, columnName)
    }

    return primaryKeys, nil
}

func (p *PostgreSQLExtractor) extractViews(cfg config.SchemaConfig) ([]schema.View, error) {
    query := `
        SELECT table_name, view_definition
        FROM information_schema.views
        WHERE table_schema = 'public'
        ORDER BY table_name
    `

    rows, err := p.db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var views []schema.View
    for rows.Next() {
        var view schema.View
        if err := rows.Scan(&view.Name, &view.Definition); err != nil {
            return nil, err
        }

        if len(cfg.IncludeTables) > 0 && !contains(cfg.IncludeTables, view.Name) {
            continue
        }
        if contains(cfg.ExcludeTables, view.Name) {
            continue
        }

        view.Schema = "public"
        
        columns, err := p.extractColumns(view.Name)
        if err != nil {
            return nil, err
        }
        view.Columns = columns

        views = append(views, view)
    }

    return views, nil
}

func (p *PostgreSQLExtractor) extractForeignKeys(cfg config.SchemaConfig) ([]schema.ForeignKey, error) {
    query := `
        SELECT
            tc.constraint_name,
            tc.table_name,
            kcu.column_name,
            ccu.table_name AS foreign_table_name,
            ccu.column_name AS foreign_column_name,
            rc.update_rule,
            rc.delete_rule
        FROM information_schema.table_constraints AS tc
        JOIN information_schema.key_column_usage AS kcu
            ON tc.constraint_name = kcu.constraint_name
        JOIN information_schema.constraint_column_usage AS ccu
            ON ccu.constraint_name = tc.constraint_name
        JOIN information_schema.referential_constraints AS rc
            ON rc.constraint_name = tc.constraint_name
        WHERE tc.constraint_type = 'FOREIGN KEY'
            AND tc.table_schema = 'public'
    `

    rows, err := p.db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var foreignKeys []schema.ForeignKey
    for rows.Next() {
        var fk schema.ForeignKey
        if err := rows.Scan(
            &fk.Name,
            &fk.Table,
            &fk.Column,
            &fk.ReferencedTable,
            &fk.ReferencedColumn,
            &fk.OnUpdate,
            &fk.OnDelete,
        ); err != nil {
            return nil, err
        }

        if len(cfg.IncludeTables) > 0 && !contains(cfg.IncludeTables, fk.Table) && !contains(cfg.IncludeTables, fk.ReferencedTable) {
            continue
        }
        if contains(cfg.ExcludeTables, fk.Table) || contains(cfg.ExcludeTables, fk.ReferencedTable) {
            continue
        }

        foreignKeys = append(foreignKeys, fk)
    }

    return foreignKeys, nil
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if strings.EqualFold(s, item) {
            return true
        }
    }
    return false
}