# Query Engine

This document covers the technical architecture of Octobud's query engine. For query syntax and usage, see the [Query Syntax Guide](../guides/query-syntax.md).

## Overview

The query engine parses user queries, validates them, and translates them into database searches or in-memory evaluations. It's designed as a multi-stage pipeline that separates parsing from execution.

## Architecture

The query engine is built as a multi-stage pipeline:

1. **Lexer** - Tokenizes query strings into individual tokens (fields, operators, values, etc.)
2. **Parser** - Builds an Abstract Syntax Tree (AST) from tokens, handling operator precedence and grouping
3. **Validator** - Validates field names and values against allowed options
4. **SQL Builder** - Generates parameterized SQL queries from the AST, adding JOINs only when needed
5. **Evaluator** - Evaluates queries in-memory for rule matching and action hints

```
Query String → Lexer → Parser → AST → Validator → SQL Builder → Database Query
                                  ↓
                              Evaluator → In-Memory Matching
```

## Design Principles

- **Pure translation** - No business logic in the parser; defaults (like inbox filtering) are handled separately
- **Reusability** - Same parser for SQL generation and in-memory evaluation
- **Efficiency** - JOINs only added when needed, query parsing happens once per request
- **Safety** - Parameterized queries prevent SQL injection

## Dual Execution Paths

The same parsed AST can be executed two ways:

### Database Queries

For fetching notifications from the database:
- AST is translated to parameterized SQL WHERE clauses
- JOINs are added only when filtering on related data (e.g., repository, tags)
- Business logic defaults are applied after parsing

### In-Memory Evaluation

For rule matching and action hints:
- AST is evaluated directly against notification objects
- Used to check if a notification matches a rule's query
- Used to predict if an action would dismiss a notification from the current view

## Query Processing Flow

1. **Parse** - Query string is tokenized and parsed into an AST
2. **Validate** - Field names and values are checked against allowed options
3. **Apply defaults** - Context-specific defaults are added (e.g., inbox excludes archived/muted)
4. **Execute** - Either generate SQL or evaluate in-memory

## Operator Precedence

The parser handles operator precedence correctly:

1. Parentheses (highest)
2. NOT
3. AND
4. OR (lowest)

This ensures queries like `a OR b AND c` are parsed as `a OR (b AND c)`.

## Error Handling

The query engine provides helpful error messages for:

- Unknown fields (e.g., `badfield:value`)
- Invalid values for operators (e.g., `in:badvalue`)
- Syntax errors (e.g., mismatched parentheses)
- Unclosed quotes

