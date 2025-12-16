# Field Symbols Example

This example demonstrates the power of TwinCAT's `{attribute 'symbol'}` pragma combined with the library's automatic type detection.

## What This Shows

When you add `{attribute 'symbol'}` to struct fields in your PLC code, you can:

1. **Read anything** - structs, fields, primitives - with one function
2. **Write directly** - to individual fields without struct manipulation
3. **Subscribe efficiently** - monitor specific fields instead of whole structs
4. **Write type-agnostic code** - handle any symbol without knowing its structure

## Prerequisites

This example requires that you add the `{attribute 'symbol'}` pragma to your PLC struct fields:

```iecst
TYPE ST_StructExample2 :
STRUCT
    {attribute 'symbol'}
    iTest : INT;
    {attribute 'symbol'}
    stTest : ST_NestedStruct;
END_STRUCT
END_TYPE

TYPE ST_NestedStruct :
STRUCT
    {attribute 'symbol'}
    iTest : INT;
    {attribute 'symbol'}
    sTest : STRING;
    {attribute 'symbol'}
    uiTest : UINT;
END_STRUCT
END_TYPE

VAR_GLOBAL
    structExample2 : ST_StructExample2;
END_VAR
```

## What Gets Demonstrated

1. **Reading Flexibility**

   - Read entire struct: `ReadSymbolValue(ctx, "MAIN.structExample2")`
   - Read specific field: `ReadSymbolValue(ctx, "MAIN.structExample2.iTest")`
   - Read nested field: `ReadSymbolValue(ctx, "MAIN.structExample2.stTest.iTest")`

2. **Direct Field Writing**

   - No more struct manipulation
   - Write directly: `WriteSymbolValue(ctx, "MAIN.structExample2.iTest", int16(7777))`

3. **Field-Level Subscriptions**

   - Monitor only what you care about
   - Subscribe: `SubscribeSymbol(ctx, "MAIN.structExample2.iTest", callback)`

4. **Type-Agnostic Operations**
   - Same code handles structs, fields, primitives
   - No need to know the type beforehand

## Running

```bash
go run ./examples/field-symbols
```

## Expected Behavior

### Without the pragma:

- Reads will fail for individual fields
- Writes will fail for individual fields
- Subscriptions will fail for individual fields
- Only whole struct operations work

### With the pragma:

- Everything works seamlessly
- All reads succeed
- All writes succeed
- All subscriptions work
- True type-agnostic programming

## Comparison

| Operation          | Without Pragma           | With Pragma             |
| ------------------ | ------------------------ | ----------------------- |
| Read whole struct  | ✅ `ReadSymbolValue()`   | ✅ `ReadSymbolValue()`  |
| Read field         | ❌ Not possible          | ✅ `ReadSymbolValue()`  |
| Write field        | ⚠️ `WriteStructFields()` | ✅ `WriteSymbolValue()` |
| Subscribe to field | ❌ Not possible          | ✅ `SubscribeSymbol()`  |

## Why This Matters

With the pragma on all fields:

- **Simpler code** - Use 4 functions for everything
- **Better performance** - Read/write/subscribe to only what you need
- **More flexible** - Build generic tools that work with any variable
- **Easier debugging** - Monitor specific fields instead of whole structs
