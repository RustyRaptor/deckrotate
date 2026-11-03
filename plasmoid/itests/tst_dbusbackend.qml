import QtQuick
import QtTest
import "../package/contents/ui"

// Runs against internal/service/testservice on a private bus (scripts/test-dbus.sh).
Item {
    id: root

    DBusBackend {
        id: backend
    }

    TestCase {
        name: "DBusBackend"

        function initTestCase() {
            tryCompare(backend, "available", true, 5000)
        }

        function test_1_initial_state() {
            tryCompare(backend, "mode", "auto")
            compare(backend.locked, false)
            compare(backend.imu, true)
            compare(backend.device, "landscape")
            compare(backend.screen, "landscape")
            compare(backend.suggested, "")
        }

        function test_2_lock() {
            backend.setLocked(true)
            tryCompare(backend, "locked", true)
            backend.setLocked(false)
            tryCompare(backend, "locked", false)
        }

        function test_3_rotate() {
            backend.rotateTo("portrait-ccw")
            tryCompare(backend, "screen", "portrait-ccw")
            backend.rotateBy(1)
            tryCompare(backend, "screen", "landscape")
            backend.rotateBy(-1)
            tryCompare(backend, "screen", "portrait-ccw")
            backend.rotateTo("landscape")
            tryCompare(backend, "screen", "landscape")
        }

        function test_4_mode() {
            backend.setMode("manual")
            tryCompare(backend, "mode", "manual")
            backend.setMode("auto")
            tryCompare(backend, "mode", "auto")
        }

        function test_5_types_are_plain() {
            compare(typeof backend.mode, "string")
            compare(typeof backend.locked, "boolean")
        }
    }
}
