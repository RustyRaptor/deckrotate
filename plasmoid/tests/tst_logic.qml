import QtQuick
import QtTest
import "../package/contents/ui/logic.js" as Logic

TestCase {
    name: "Logic"

    function test_orientations_order() {
        compare(Logic.orientations, ["landscape", "portrait-cw", "landscape-flipped", "portrait-ccw"])
    }

    function test_labels_data() {
        return [
            { tag: "landscape", o: "landscape", want: "Landscape" },
            { tag: "portrait-cw", o: "portrait-cw", want: "Portrait (right grip down)" },
            { tag: "flipped", o: "landscape-flipped", want: "Landscape (flipped)" },
            { tag: "portrait-ccw", o: "portrait-ccw", want: "Portrait (left grip down)" },
            { tag: "unknown", o: "unknown", want: "Unknown" },
            { tag: "empty", o: "", want: "Unknown" }
        ]
    }

    function test_labels(data) {
        compare(Logic.label(data.o), data.want)
    }

    function test_angles() {
        compare(Logic.orientations.map(Logic.angle), [0, 90, 180, 270])
        compare(Logic.angle("unknown"), 0)
    }

    function test_icon() {
        compare(Logic.iconName({ available: true, locked: false }), "transform-rotate-symbolic")
        compare(Logic.iconName({ available: true, locked: true }), "object-locked-symbolic")
        compare(Logic.iconName({ available: false, locked: false }), "transform-rotate-symbolic")
    }

    function test_summary_data() {
        return [
            { tag: "down", b: { available: false }, want: "Rotation service is not running" },
            { tag: "auto", b: { available: true, imu: true, locked: false, mode: "auto", screen: "landscape" }, want: "Auto-rotate · Landscape" },
            { tag: "manual", b: { available: true, imu: true, locked: false, mode: "manual", screen: "portrait-ccw" }, want: "Manual · Portrait (left grip down)" },
            { tag: "locked", b: { available: true, imu: true, locked: true, mode: "auto", screen: "landscape-flipped" }, want: "Locked · Landscape (flipped)" },
            { tag: "no-imu", b: { available: true, imu: false, locked: false, mode: "auto", screen: "landscape" }, want: "No orientation sensor · Landscape" }
        ]
    }

    function test_summary(data) {
        compare(Logic.summary(data.b), data.want)
    }
}
