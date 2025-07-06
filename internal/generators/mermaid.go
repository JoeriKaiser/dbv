package generators

import (
	"dbv/internal/schema"
	"fmt"
	"strings"
)

func GenerateMermaid(s *schema.Schema) string {
    var builder strings.Builder

    builder.WriteString("# Database Schema Diagram\n\n")
    builder.WriteString("```mermaid\nerDiagram\n")

    for _, table := range s.Tables {
        builder.WriteString(fmt.Sprintf("    %s {\n", cleanTableName(table.Name)))
        
        for _, col := range table.Columns {
            typeStr := formatMermaidType(col)
            keyStr := ""
            if col.IsPrimaryKey {
                keyStr = " PK"
            } else if !col.IsNullable {
                keyStr = " NOT NULL"
            }
            
            builder.WriteString(fmt.Sprintf("        %s %s%s\n", typeStr, col.Name, keyStr))
        }
        
        builder.WriteString("    }\n\n")
    }

    for _, view := range s.Views {
        builder.WriteString(fmt.Sprintf("    %s {\n", cleanTableName(view.Name)))
        
        for _, col := range view.Columns {
            typeStr := formatMermaidType(col)
            builder.WriteString(fmt.Sprintf("        %s %s\n", typeStr, col.Name))
        }
        
        builder.WriteString("    }\n\n")
    }

    for _, fk := range s.ForeignKeys {
        relationship := determineRelationship(fk)
        builder.WriteString(fmt.Sprintf("    %s %s %s : %s\n", 
            cleanTableName(fk.ReferencedTable), 
            relationship, 
            cleanTableName(fk.Table), 
            fk.Column))
    }

    builder.WriteString("```\n\n")
    builder.WriteString(fmt.Sprintf("Generated on: %s\n", s.GeneratedAt.Format("2006-01-02 15:04:05")))
    builder.WriteString(fmt.Sprintf("Total Tables: %d\n", len(s.Tables)))
    builder.WriteString(fmt.Sprintf("Total Views: %d\n", len(s.Views)))
    builder.WriteString(fmt.Sprintf("Total Foreign Keys: %d\n", len(s.ForeignKeys)))

    return builder.String()
}

func formatMermaidType(col schema.Column) string {
    switch strings.ToLower(col.Type) {
    case "varchar", "text", "char", "string":
        if col.Length != nil {
            return fmt.Sprintf("varchar(%d)", *col.Length)
        }
        return "varchar"
    case "int", "integer", "bigint":
        return "int"
    case "decimal", "numeric":
        if col.Precision != nil && col.Scale != nil {
            return fmt.Sprintf("decimal(%d,%d)", *col.Precision, *col.Scale)
        }
        return "decimal"
    case "boolean", "bool":
        return "boolean"
    case "date":
        return "date"
    case "timestamp", "datetime":
        return "timestamp"
    default:
        return col.Type
    }
}

func cleanTableName(name string) string {
    name = strings.ReplaceAll(name, "-", "_")
    name = strings.ReplaceAll(name, ".", "_")
    return name
}

func determineRelationship(fk schema.ForeignKey) string {
    return "||--o{"
}