package database

import (
	"database/sql"
	"dbv/internal/schema"
	"dbv/pkg/config"
	"fmt"
	"time"
)

type SQLiteExtractor struct {
    db *sql.DB
}

func (s *SQLiteExtractor) ExtractSchema(cfg config.SchemaConfig) (*schema.Schema, error) {
    sch := &schema.Schema{
        Database:    "sqlite",
        GeneratedAt: time.Now(),
    }

    tables, err := s.extractTables(cfg)
    if err != nil {
        return nil, err
    }
    sch.Tables = tables

    if cfg.IncludeViews {
        views, err := s.extractViews(cfg)
        if err != nil {
            return nil, err
        }
        sch.Views = views
    }

    foreignKeys, err := s.extractForeignKeys(cfg)
    if err != nil {
        return nil, err
    }
    sch.ForeignKeys = foreignKeys

    return sch, nil
}

func (s *SQLiteExtractor) extractTables(cfg config.SchemaConfig) ([]schema.Table, error) {
    query := `
        SELECT name, type, sql
        FROM sqlite_master
        WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
        ORDER BY name
    `

    rows, err := s.db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var tables []schema.Table
    for rows.Next() {
        var table schema.Table
        var sqlDef sql.NullString
        if err := rows.Scan(&table.Name, &table.Type, &sqlDef); err != nil {
            return nil, err
        }

        if len(cfg.IncludeTables) > 0 && !contains(cfg.IncludeTables, table.Name) {
            continue
        }
        if contains(cfg.ExcludeTables, table.Name) {
            continue
        }

        table.Schema = "main"
        table.Type = "BASE TABLE"

        columns, err := s.extractColumns(table.Name)
        if err != nil {
            return nil, err
        }
        table.Columns = columns

        primaryKeys, err := s.extractPrimaryKeys(table.Name)
        if err != nil {
            return nil, err
        }
        table.PrimaryKeys = primaryKeys

        tables = append(tables, table)
    }

    return tables, nil
}

func (s *SQLiteExtractor) extractColumns(tableName string) ([]schema.Column, error) {
    query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)

    rows, err := s.db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var columns []schema.Column
    for rows.Next() {
        var col schema.Column
        var cid int
        var defaultValue sql.NullString
        var notNull int
        var pk int

        if err := rows.Scan(
            &cid,
            &col.Name,
            &col.Type,
            &notNull,
            &defaultValue,
            &pk,
        ); err != nil {
            return nil, err
        }

        col.IsNullable = notNull == 0
        col.IsPrimaryKey = pk == 1
        if defaultValue.Valid {
            col.DefaultValue = &defaultValue.String
        }

        columns = append(columns, col)
    }

    return columns, nil
}

func (s *SQLiteExtractor) extractPrimaryKeys(tableName string) ([]string, error) {
    query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)

    rows, err := s.db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var primaryKeys []string
    for rows.Next() {
        var cid int
        var name, dataType string
        var notNull int
        var defaultValue sql.NullString
        var pk int

        if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
            return nil, err
        }

        if pk == 1 {
            primaryKeys = append(primaryKeys, name)
        }
    }

    return primaryKeys, nil
}

func (s *SQLiteExtractor) extractViews(cfg config.SchemaConfig) ([]schema.View, error) {
    query := `
        SELECT name, sql
        FROM sqlite_master
        WHERE type = 'view'
        ORDER BY name
    `

    rows, err := s.db.Query(query)
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

        view.Schema = "main"

        columns, err := s.extractColumns(view.Name)
        if err != nil {
            return nil, err
        }
        view.Columns = columns

        views = append(views, view)
    }

    return views, nil
}

func (s *SQLiteExtractor) extractForeignKeys(cfg config.SchemaConfig) ([]schema.ForeignKey, error) {
    tables, err := s.extractTables(cfg)
    if err != nil {
        return nil, err
    }

    var foreignKeys []schema.ForeignKey
    for _, table := range tables {
        query := fmt.Sprintf("PRAGMA foreign_key_list(%s)", table.Name)

        rows, err := s.db.Query(query)
        if err != nil {
            return nil, err
        }

        for rows.Next() {
            var fk schema.ForeignKey
            var id, seq int
            var onUpdate, onDelete, match string

            if err := rows.Scan(
                &id,
                &seq,
                &fk.ReferencedTable,
                &fk.Column,
                &fk.ReferencedColumn,
                &onUpdate,
                &onDelete,
                &match,
            ); err != nil {
                rows.Close()
                return nil, err
            }

            fk.Name = fmt.Sprintf("fk_%s_%s", table.Name, fk.Column)
            fk.Table = table.Name
            fk.OnUpdate = onUpdate
            fk.OnDelete = onDelete

            if len(cfg.IncludeTables) > 0 && !contains(cfg.IncludeTables, fk.Table) && !contains(cfg.IncludeTables, fk.ReferencedTable) {
                continue
            }
            if contains(cfg.ExcludeTables, fk.Table) || contains(cfg.ExcludeTables, fk.ReferencedTable) {
                continue
            }

            foreignKeys = append(foreignKeys, fk)
        }
        rows.Close()
    }

    return foreignKeys, nil
}