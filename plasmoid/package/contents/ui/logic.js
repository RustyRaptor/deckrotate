.pragma library

const pluginId = "io.github.rustyraptor.deckrotate"
const domain = "plasma_applet_" + pluginId

// Clockwise quarter turns from normal landscape, matching the daemon's names.
const orientations = ["landscape", "portrait-cw", "landscape-flipped", "portrait-ccw"]

const labels = {
    "landscape": "Landscape",
    "portrait-cw": "Portrait (right grip down)",
    "landscape-flipped": "Landscape (flipped)",
    "portrait-ccw": "Portrait (left grip down)"
}

function label(o) {
    return labels[o] ?? "Unknown"
}

function angle(o) {
    const i = orientations.indexOf(o)
    return i < 0 ? 0 : i * 90
}

function iconName(b) {
    return b.available && b.locked ? "object-locked-symbolic" : "transform-rotate-symbolic"
}

function summary(b) {
    if (!b.available)
        return "Rotation service is not running"
    let state
    if (b.locked)
        state = "Locked"
    else if (!b.imu)
        state = "No orientation sensor"
    else
        state = b.mode === "manual" ? "Manual" : "Auto-rotate"
    return state + " · " + label(b.screen)
}
