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

// Mirrors the daemon's limits (internal/core); the daemon has the final say.
const settleRange = { from: 100, to: 3000, step: 100 }
const promptRange = { from: 2000, to: 30000, step: 1000 }

function seconds(ms) {
    const s = ms / 1000
    return (Number.isInteger(s) ? String(s) : s.toFixed(1)) + " s"
}

function parseSeconds(text) {
    const m = String(text).trim().replace(",", ".").match(/^([0-9]+(?:\.[0-9]+)?)\s*s?$/)
    return m ? Math.round(parseFloat(m[1]) * 1000) : NaN
}
