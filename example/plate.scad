// Plate with configurable dimensions
// This SCAD file demonstrates a simple plate that can be configured

// Use config files if they exist
use <./cfg.scad>

module plate() {
    width = plate_width();
    height = plate_height();
    thickness = plate_thickness();
    
    // Simple rectangular plate
    cube([width, height, thickness]);
}

plate();