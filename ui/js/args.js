.pragma library

// Parses the overlay's command line: --vertical top|bottom --horizontal left|right --angle 0|90|180|270
function parse(argv) {
    const out = { vertical: "bottom", horizontal: "right", angle: 0 }
    for (let i = 0; i + 1 < argv.length; i++) {
        const v = argv[i + 1]
        switch (argv[i]) {
        case "--vertical":
            if (v === "top" || v === "bottom") out.vertical = v
            break
        case "--horizontal":
            if (v === "left" || v === "right") out.horizontal = v
            break
        case "--angle": {
            const n = Number(v)
            if ([0, 90, 180, 270].includes(n)) out.angle = n
            break
        }
        }
    }
    return out
}

// The Deck's built-in panel, so the button never lands on an external monitor.
function panelScreen(screens) {
    return screens.find(s => s.name.startsWith("eDP")) ?? screens[0] ?? null
}

function edges(opts) {
    return {
        top: opts.vertical === "top",
        bottom: opts.vertical === "bottom",
        left: opts.horizontal === "left",
        right: opts.horizontal === "right"
    }
}
