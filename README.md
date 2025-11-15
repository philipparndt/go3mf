# go3mf

A command-line tool for working with 3D model files. It can render OpenSCAD files and combine multiple 3D model files (3MF, STL, SCAD) into a single 3MF output file.

## Install

```bash
brew install philipparndt/go3mf/go3mf
```

## Commands

### combine (alias: build)

Combine files into a single 3MF file. This command intelligently handles different file types:
- **YAML config files** - Use structured configuration for complex multi-object models
- **SCAD files** - Render OpenSCAD files and combine them
- **3MF files** - Merge existing 3MF models
- **STL files** - Convert STL meshes (ASCII and binary) to 3MF and combine them

```bash
go3mf combine [OPTIONS] <files...>
# or use the 'build' alias
go3mf build [OPTIONS] <files...>
```

**Options:**
- `-o, --output` - Output file path (default: "combined.3mf")
- `--object` - Define an object group for SCAD files (can be repeated)

**Note:** The `build` command is an alias for `combine` and works identically.

---

#### Using YAML Configuration (Recommended for Complex Models)

For complex models with multiple objects and parts, use a YAML configuration file. This is the recommended way as it provides better organization and reusability.

```bash
go3mf combine example/config.yaml
```

**YAML Configuration Format:**

```yaml
# Output file path (relative to config file or absolute)
output: combined.3mf

# Define objects - each object groups related parts
objects:
  # Single-part object
  - name: Base
    parts:
      - name: platform
        file: base.scad
        filament: 1  # Optional: 1-4 for AMS slots, 0 or omit for auto

  # Multi-part object
  - name: Assembly
    parts:
      - name: main_body
        file: body.scad
        filament: 2
      
      - name: cover
        file: cover.scad
        filament: 3
```

**Configuration Fields:**
- `output` - Output 3MF file path (required)
- `objects` - Array of objects (required, at least one)
  - `name` - Object name (required)
  - `config` - Array of config files (optional, can be at object or part level)
  - `parts` - Array of parts in the object (required, at least one)
    - `name` - Part name (required)
    - `file` - Path to SCAD file, relative to config or absolute (required)
    - `filament` - AMS filament slot: 0=auto, 1-4=specific slot (optional)
    - `config` - Array of config files for this part (optional)

**SCAD Configuration Files:**

You can pass configuration values to OpenSCAD files using config sections. Two formats are supported:

**New Map-Based Format (Recommended):**
```yaml
objects:
  - name: Holder
    parts:
      - name: custom_holder
        file: holder.scad
        config:
          - cfg.scad:
              h: 6
              width: 38
              length: 90
              diameter: 16.5
              name: "Custom Holder"
              rounded: true
```

This automatically generates SCAD functions:
```openscad
function get_h() = 6;
function get_width() = 38;
function get_length() = 90;
function get_diameter() = 16.5;
function get_name() = "Custom Holder";
function get_rounded() = true;
```

**Old String Format (Still Supported):**
```yaml
config:
  - cfg.scad: |
      function get_h() = 6;
      function get_width() = 38;
      function get_length() = 90;
```

**Config Features:**
- Supports integers, floats, strings, and booleans
- Strings are automatically quoted
- Object-level config applies to all parts
- Part-level config overrides object-level config
- Both formats can be mixed in the same file

**Benefits:**
- Organize complex models with multiple objects and parts
- Reusable configuration files for reproducible builds
- Clear structure for multi-color prints with AMS
- File paths relative to config file for portability
- Version control friendly
- Parameterize SCAD files with clean config syntax

**Examples:**
```bash
# Use the example configuration
go3mf combine example/config.yaml

# Example with SCAD config files
go3mf combine example/plate-config.yaml

# Complete config formats demo
go3mf combine example/config-formats-demo.yaml
```

See `example/config.yaml`, `example/plate-config.yaml`, and `example/config-formats-demo.yaml` for complete examples.

---

#### Combining SCAD Files

Render OpenSCAD (.scad) files and combine them into a single 3MF file.

**Simple Mode - Flat List:**

Combine SCAD files as parts in a single object:

```bash
go3mf combine file1.scad file2.scad file3.scad -o output.3mf
```

This creates a single object named "Combined" with multiple parts.

**File argument formats:**
- `file.scad` - Use filename as part name, auto-assign filament
- `file.scad:name` - Custom part name, auto-assign filament  
- `file.scad:name:2` - Custom name with specific filament slot (1-4)

Examples:
```bash
# Basic combination with auto-assigned filaments
go3mf combine button.scad holder.scad base.scad -o parts.3mf

# Specify custom names
go3mf combine button.scad:btn holder.scad:holder -o output.3mf

# Manually assign specific filament slots (AMS slots 1-4)
go3mf combine button.scad:button:1 holder.scad:holder:2 base.scad:base:3
```

**Advanced Mode - Object Grouping (Recommended):**

Use the `--object` flag with `-n` (name) and `-c` (color/filament) for better organization and tab completion:

```bash
go3mf combine -o output.3mf \
  --object -n "ObjectName" -c 1 file1.scad -c 2 file2.scad \
  --object -n "NextObject" -c 3 file3.scad
```

- `--object` - Start a new object group
- `-n "Name"` - Set the object name (required after --object)
- `-c N` - Set filament slot (1-4) for the next file (optional)
- Files support tab completion!

Examples:
```bash
# Group files into objects with specific filaments
go3mf combine -o assembly.3mf \
  --object -n "Case" -c 1 bottom.scad -c 1 top.scad \
  --object -n "Inserts" -c 2 insert1.scad -c 2 insert2.scad

# Mix with auto-filament (omit -c flag)
go3mf combine -o model.3mf \
  --object -n "Body" body.scad cover.scad \
  --object -n "Accessories" -c 4 button.scad

# Use relative or absolute paths with tab completion
go3mf combine -o project.3mf \
  --object -n "Main" -c 1 ./parts/base.scad ./parts/frame.scad \
  --object -n "Details" -c 3 ./details/badge.scad
```

**Filament Assignment:**
When combining multiple objects, go3mf automatically assigns different filament slots for Bambu Studio:
- If no filament slot is specified (no `-c` flag), objects are automatically assigned slots 1, 2, 3, 4 (cycling)
- You can manually specify slots 1-4 to control which AMS filament each part uses
- This allows Bambu Studio to recognize different filaments per object without manual configuration

---

#### Combining 3MF Files

Combine multiple existing 3MF files into a single 3MF model.

```bash
go3mf combine file1.3mf file2.3mf file3.3mf -o result.3mf
```

**Filament Assignment:**
Objects are automatically assigned different filament slots (1-4) for Bambu Studio, cycling through available AMS slots.

---

#### Combining STL Files

Convert and combine multiple STL files into a single 3MF model. STL files (both ASCII and binary formats) are automatically converted to 3MF format and then combined.

```bash
go3mf combine file1.stl file2.stl file3.stl -o combined.3mf
```

**Note:** The output file must have a `.3mf` extension as STL files are converted and embedded into the 3MF format.

---

### inspect

Inspect a 3MF file and display its contents, including objects, parts, colors, and metadata. This is useful for understanding the structure of 3MF files and verifying the output of combine operations.

```bash
go3mf inspect <file.3mf>
```

**What it shows:**
- Basic file information (unit, language)
- Metadata (application, creation date, etc.)
- Build plate items (what objects are printable)
- Object hierarchy with components and parts
- Color/filament assignments (when available)
- Object and part names

**Examples:**

```bash
# Inspect a combined 3MF file
go3mf inspect combined.3mf

# Inspect any 3MF file to see its structure
go3mf inspect model.3mf
```

**Sample output:**

```
Inspecting: combined_from_yaml.3mf
  • Unit: millimeter
  • Language: 
  • Metadata:
  •   - Application: go3mf
  •   - BambuStudio:3mfVersion: 1
  •   - CreationDate: 2025-11-07
  •   - ModificationDate: 2025-11-07
Build Plate Items:
  • 1. Object ID 1: Base (printable: yes)
  • 2. Object ID 4: Assembly (printable: yes) [offset: 90.00, 0.00, 0.00]
Objects in Model:
  • • Base (ID: 1) (filament: 1) [has mesh]
  • • Assembly (ID: 4) - 2 part(s) (filament: 1)
  •   - Assembly/main_body (ID: 2) (filament: 2)
  •   - Assembly/cover (ID: 3) (filament: 3)
```

This shows the same object/part structure that's displayed after a combine operation, making it easy to verify your 3MF files.

---

### version

Display version information.

```bash
go3mf version
```
