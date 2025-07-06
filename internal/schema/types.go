package schema

import "time"

type Schema struct {
    Database    string       `json:"database"`
    Tables      []Table      `json:"tables"`
    Views       []View       `json:"views"`
    ForeignKeys []ForeignKey `json:"foreign_keys"`
    Indexes     []Index      `json:"indexes"`
    GeneratedAt time.Time    `json:"generated_at"`
}

type Table struct {
    Name        string   `json:"name"`
    Schema      string   `json:"schema"`
    Type        string   `json:"type"`
    Columns     []Column `json:"columns"`
    PrimaryKeys []string `json:"primary_keys"`
    Comment     string   `json:"comment"`
}

type View struct {
    Name       string   `json:"name"`
    Schema     string   `json:"schema"`
    Definition string   `json:"definition"`
    Columns    []Column `json:"columns"`
    Comment    string   `json:"comment"`
}

type Column struct {
    Name         string  `json:"name"`
    Type         string  `json:"type"`
    Length       *int    `json:"length,omitempty"`
    Precision    *int    `json:"precision,omitempty"`
    Scale        *int    `json:"scale,omitempty"`
    IsNullable   bool    `json:"is_nullable"`
    DefaultValue *string `json:"default_value,omitempty"`
    IsPrimaryKey bool    `json:"is_primary_key"`
    IsUnique     bool    `json:"is_unique"`
    Comment      string  `json:"comment"`
}

type ForeignKey struct {
    Name             string `json:"name"`
    Table            string `json:"table"`
    Column           string `json:"column"`
    ReferencedTable  string `json:"referenced_table"`
    ReferencedColumn string `json:"referenced_column"`
    OnUpdate         string `json:"on_update"`
    OnDelete         string `json:"on_delete"`
}

type Index struct {
    Name      string   `json:"name"`
    Table     string   `json:"table"`
    Columns   []string `json:"columns"`
    IsUnique  bool     `json:"is_unique"`
    IsPrimary bool     `json:"is_primary"`
    Type      string   `json:"type"`
}