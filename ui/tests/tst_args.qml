import QtQuick
import QtTest
import "../js/args.js" as Args

TestCase {
    name: "OverlayArgs"

    function test_defaults() {
        const o = Args.parse(["/usr/lib/qt6/bin/qml", "overlay.qml", "--"])
        compare(o.vertical, "bottom")
        compare(o.horizontal, "right")
        compare(o.angle, 0)
    }

    function test_parse() {
        const o = Args.parse(["qml", "overlay.qml", "--", "--vertical", "top", "--horizontal", "left", "--angle", "270"])
        compare(o.vertical, "top")
        compare(o.horizontal, "left")
        compare(o.angle, 270)
    }

    function test_rejects_junk_data() {
        return [
            { tag: "vertical", args: ["--vertical", "middle"], key: "vertical", want: "bottom" },
            { tag: "horizontal", args: ["--horizontal", "up"], key: "horizontal", want: "right" },
            { tag: "angle", args: ["--angle", "45"], key: "angle", want: 0 },
            { tag: "angle-nan", args: ["--angle", "lots"], key: "angle", want: 0 },
            { tag: "dangling", args: ["--angle"], key: "angle", want: 0 }
        ]
    }

    function test_rejects_junk(data) {
        compare(Args.parse(data.args)[data.key], data.want)
    }

    function test_panel_screen() {
        const ext = { name: "DP-1" }
        const panel = { name: "eDP-1" }
        compare(Args.panelScreen([ext, panel]), panel)
        compare(Args.panelScreen([ext]), ext)
        compare(Args.panelScreen([]), null)
    }

    function test_anchor_edges() {
        compare(Args.edges({ vertical: "top", horizontal: "left" }), { top: true, bottom: false, left: true, right: false })
        compare(Args.edges({ vertical: "bottom", horizontal: "right" }), { top: false, bottom: true, left: false, right: true })
    }
}
