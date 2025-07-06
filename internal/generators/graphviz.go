package generators

import (
	"dbv/internal/schema"
	"fmt"
	"strings"
)

func GenerateGraphviz(s *schema.Schema) string {
    var builder strings.Builder

    builder.WriteString("digraph schema {\n")
    builder.WriteString("  rankdir=TB;\n")
    builder.WriteString("  node [shape=record, style=filled, fillcolor=lightblue];\n")
    builder.WriteString("  edge [color=gray];\n\n")

    for _, table := range s.Tables {
        builder.WriteString(fmt.Sprintf("  %s [label=\"{%s|", cleanNodeName(table.Name), table.Name))
        
        var fields []string
        for _, col := range table.Columns {
            field := col.Name + ": " + formatGraphvizType(col)
            if col.IsPrimaryKey {
                field = "+" + field
            }
            if !col.IsNullable {
                field += " NOT NULL"
            }
            fields = append(fields, field)
        }
        
        builder.WriteString(strings.Join(fields, "\\l"))
        builder.WriteString("\\l}\"];\n")
    }

    // Generate views
    for _, view := range s.Views {
        builder.WriteString(fmt.Sprintf("  %s [label=\"{%s (VIEW)|", cleanNodeName(view.Name), view.Name))
        
        var fields []string
        for _, col := range view.Columns {
            field := col.Name + ": " + formatGraphvizType(col)
            fields = append(fields, field)
        }
        
        builder.WriteString(strings.Join(fields, "\\l"))
        builder.WriteString("\\l}\", fillcolor=lightgreen];\n")
    }

    builder.WriteString("\n")

    for _, fk := range s.ForeignKeys {
        builder.WriteString(fmt.Sprintf("  %s -> %s [label=\"%s\"];\n", 
            cleanNodeName(fk.ReferencedTable), 
            cleanNodeName(fk.Table), 
            fk.Column))
    }

    builder.WriteString("}\n")

    return builder.String()
}

func formatGraphvizType(col schema.Column) string {
    switch strings.ToLower(col.Type) {
    case "varchar", "text", "char", "string":
        if col.Length != nil {
            return fmt.Sprintf("VARCHAR(%d)", *col.Length)
        }
        return "VARCHAR"
    case "int", "integer":
        return "INT"
    case "bigint":
        return "BIGINT"
    case "decimal", "numeric":
        if col.Precision != nil && col.Scale != nil {
            return fmt.Sprintf("DECIMAL(%d,%d)", *col.Precision, *col.Scale)
        }
        return "DECIMAL"
    case "boolean", "bool":
        return "BOOL"
    case "date":
        return "DATE"
    case "timestamp", "datetime":
        return "TIMESTAMP"
    default:
        return strings.ToUpper(col.Type)
    }
}

func cleanNodeName(name string) string {
    name = strings.ReplaceAll(name, "-", "_")
    name = strings.ReplaceAll(name, ".", "_")
    name = strings.ReplaceAll(name, " ", "_")
    return name
}