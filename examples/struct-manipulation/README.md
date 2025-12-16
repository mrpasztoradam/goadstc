# Struct Manipulation Example

This example demonstrates how to read and write to struct fields using automatic type detection and encoding.

## What it does

1. **Reads `MAIN.structExample2`** using automatic type detection

   - Discovers all fields and their types automatically
   - No manual type definitions needed

2. **Writes to all struct fields** based on detected types

   - Automatically determines appropriate values for each field type
   - Handles all common TwinCAT types (BOOL, INT, UINT, REAL, STRING, TIME, etc.)
   - Skips complex types (nested structs, arrays) with informative messages

3. **Verifies the writes** by reading the struct again

## Key Features

- **Automatic Type Detection**: Uses `ReadSymbolValue()` to automatically parse the struct
- **Type-Based Writing**: Determines write values based on detected Go types
- **Comprehensive Type Support**: Handles all basic TwinCAT types
- **Dot Notation**: Accesses struct fields using `MAIN.structExample2.fieldName` syntax

## Usage

```bash
go run main.go
```

## Output Example

```
╔══════════════════════════════════════════════════════════╗
║    Struct Manipulation Example                          ║
╚══════════════════════════════════════════════════════════╝

🔌 Connecting to PLC at 10.10.0.3:48898...
✅ Connected successfully

═══════════════════════════════════════════════════════════
📖 Step 1: Reading MAIN.structExample2 with Auto-Detection
═══════════════════════════════════════════════════════════
✅ Successfully read struct with 5 fields:
  bEnabled: true (type: bool)
  nCounter: 42 (type: int16)
  fValue: 3.14 (type: float32)
  sMessage: Hello (type: string)
  tTime: 1s (type: time.Duration)

═══════════════════════════════════════════════════════════
✍️  Step 2: Writing to Struct Fields
═══════════════════════════════════════════════════════════
Writing to fields based on detected types:
  bEnabled (BOOL): writing false
    ✅ Write successful
  nCounter (INT16): writing 1234
    ✅ Write successful
  fValue (REAL): writing 3.14159
    ✅ Write successful
  sMessage (STRING): writing "Updated_sMessage"
    ✅ Write successful
  tTime (TIME): writing 500ms
    ✅ Write successful

═══════════════════════════════════════════════════════════
🔍 Step 3: Verifying Writes
═══════════════════════════════════════════════════════════
✅ Current struct values after write:
  bEnabled: false
  nCounter: 1234
  fValue: 3.14159
  sMessage: Updated_sMessage
  tTime: 500ms

✅ All operations completed successfully!
```

## Supported Types

The example handles all basic TwinCAT types:

- **Boolean**: BOOL
- **Integers**: INT8, UINT8, INT16, UINT16, INT32, UINT32, INT64, UINT64
- **Floating Point**: REAL (float32), LREAL (float64)
- **Strings**: STRING
- **Time**: TIME (time.Duration), DATE, DATE_AND_TIME (time.Time)

Complex types (nested structs and arrays) are detected and skipped with informative messages.

## How It Works

The example uses the automatic type detection feature to:

1. Read the entire struct without knowing its structure beforehand
2. Iterate over all discovered fields
3. Use Go type assertions to determine each field's type
4. Write appropriate values based on the detected types
5. Verify all writes succeeded

This approach eliminates the need for:

- Manual struct definitions in Go
- Hardcoded field names and types
- Type-specific read/write logic for each field

## Related Examples

- `examples/autotype/` - General automatic type detection examples
- `examples/typesafe/` - Type-safe read/write operations
- `examples/structs/` - Manual struct handling with type definitions
