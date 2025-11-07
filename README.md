# go3mf

A command-line tool for working with 3D model files. It can render OpenSCAD files and combine multiple 3D model files (3MF, STL, SCAD) into a single output file.

## Install

```bash
brew install philipparndt/go3mf/go3mf
```

## Commands

### combine-yaml

Combine files based on a YAML configuration file. This is the recommended way to work with multiple objects and parts, as it provides better organization and reusability.

```bash
go3mf combine-yaml <config.yaml>
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
  - `parts` - Array of parts in the object (required, at least one)
    - `name` - Part name (required)
    - `file` - Path to SCAD file, relative to config or absolute (required)
    - `filament` - AMS filament slot: 0=auto, 1-4=specific slot (optional)

**Benefits:**
- Organize complex models with multiple objects and parts
- Reusable configuration files for reproducible builds
- Clear structure for multi-color prints with AMS
- File paths relative to config file for portability
- Version control friendly

**Example:**
```bash
# Use the example configuration
go3mf combine-yaml example/config.yaml

# Create your own config and use it
go3mf combine-yaml my-project/build-config.yaml
```

See `example/config.yaml` for a complete example.

### combine-scad

Render OpenSCAD (.scad) files and combine them into a single 3MF file.

```bash
go3mf combine-scad [OPTIONS] <file1.scad> [file2.scad:name:filament] ...
```

**Options:**
- `-o, --output` - Output 3MF file path (default: "combined.3mf")

**File Format:**
- `file.scad` - Use filename as object name, auto-assign filament
- `file.scad:name` - Custom object name, auto-assign filament
- `file.scad:name:2` - Custom name with specific filament slot (1-4)

**Filament Assignment:**
When combining multiple objects, go3mf automatically assigns different filament slots for Bambu Studio:
- If no filament slot is specified, objects are automatically assigned slots 1, 2, 3, 4 (cycling)
- You can manually specify slots 1-4 to control which AMS filament each object uses
- This allows Bambu Studio to recognize different filaments per object without manual configuration

**Examples:**
```bash
go run . -/example/a.scad /example/b.scad
```

```bash
# Combine multiple SCAD files with auto-assigned filaments (1, 2, 3)
go3mf combine-scad button.scad holder.scad base.scad

# Specify custom names for objects (filaments auto-assigned)
go3mf combine-scad button.scad:button holder.scad:holder -o output.3mf

# Manually assign specific filament slots
go3mf combine-scad button.scad:button:1 holder.scad:holder:2 base.scad:base:3

# Mix auto and manual assignment
go3mf combine-scad button.scad:button:2 holder.scad:holder base.scad:base:4
```

### combine-3mf

Combine multiple 3MF files into a single 3MF model.

```bash
go3mf combine-3mf [OPTIONS] <file1.3mf> <file2.3mf> ...
```

**Options:**
- `-o, --output` - Output 3MF file path (default: "combined.3mf")

**Filament Assignment:**
Objects are automatically assigned different filament slots (1-4) for Bambu Studio, cycling through available AMS slots.

**Example:**
```bash
# Combine 3MF files with automatic filament assignment
go3mf combine-3mf model1.3mf model2.3mf model3.3mf -o result.3mf
```

### combine-stl

Combine multiple STL files into a single STL model.

```bash
go3mf combine-stl [OPTIONS] <file1.stl> <file2.stl> ...
```

**Options:**
- `-o, --output` - Output STL file path (default: "combined.stl")

**Example:**
```bash
go3mf combine-stl part1.stl part2.stl part3.stl -o combined.stl
```

### version

Display version information.

```bash
go3mf version
```

