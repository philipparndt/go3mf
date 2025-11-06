# go3mf

A command-line tool for working with 3D model files. It can render OpenSCAD files and combine multiple 3D model files (3MF, STL, SCAD) into a single output file.

## Install

```bash
brew install philipparndt/go3mf/go3mf
```

## Commands

### combine-scad

Render OpenSCAD (.scad) files and combine them into a single 3MF file.

```bash
go3mf combine-scad [OPTIONS] <file1.scad> [file2.scad:name] ...
```

**Options:**
- `-o, --output` - Output 3MF file path (default: "combined.3mf")

**Examples:**
```bash
# Combine multiple SCAD files
go3mf combine-scad button.scad holder.scad base.scad

# Specify custom names for objects
go3mf combine-scad button.scad:button holder.scad:holder -o output.3mf
```

### combine-3mf

Combine multiple 3MF files into a single 3MF model.

```bash
go3mf combine-3mf [OPTIONS] <file1.3mf> <file2.3mf> ...
```

**Options:**
- `-o, --output` - Output 3MF file path (default: "combined.3mf")

**Example:**
```bash
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

