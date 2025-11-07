// Text element with configurable content
// This SCAD file demonstrates rendering text that can be configured

// Use config files if they exist
use <./cfg.scad>

module text_element() {
    content = text_content();
    size = text_size();
    height = text_height();
    font = "Arial:style=Bold";
    
    // Get plate dimensions for positioning
    plate_w = plate_width();
    plate_h = plate_height();
    plate_t = plate_thickness();
    
    // Position text on top of plate, centered
    translate([plate_w/2, plate_h/2, plate_t]) {
        // Render 3D text
        linear_extrude(height=height)
            text(content, size=size, font=font, halign="center", valign="center");
    }
}

text_element();