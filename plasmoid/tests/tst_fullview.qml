import QtQuick
import QtTest
import "../package/contents/ui"

Item {
    id: root
    width: 420
    height: 560

    // The mock lives inside the view so teardown never leaves it dangling.
    Component {
        id: viewComponent
        FullView {
            anchors.fill: parent
            backend: MockBackend {}
        }
    }

    TestCase {
        name: "FullView"
        when: windowShown

        function make(props) {
            const view = createTemporaryObject(viewComponent, root)
            verify(view)
            for (const k in props || {})
                view.backend[k] = props[k]
            return { backend: view.backend, view: view }
        }

        function child(view, name) {
            const c = findChild(view, name)
            verify(c, "missing " + name)
            return c
        }

        function test_unavailable_shows_placeholder() {
            const t = make({ available: false })
            verify(child(t.view, "notRunning").visible)
            verify(!child(t.view, "controls").visible)
        }

        function test_available_hides_placeholder() {
            const t = make()
            verify(!child(t.view, "notRunning").visible)
            verify(child(t.view, "controls").visible)
        }

        function test_current_orientation_checked() {
            const t = make({ screen: "portrait-ccw" })
            verify(child(t.view, "orientation-portrait-ccw").checked)
            verify(!child(t.view, "orientation-landscape").checked)
            t.backend.screen = "landscape"
            verify(child(t.view, "orientation-landscape").checked)
            verify(!child(t.view, "orientation-portrait-ccw").checked)
        }

        function test_orientation_buttons_rotate_data() {
            return ["landscape", "portrait-cw", "landscape-flipped", "portrait-ccw"].map(o => ({ tag: o, o: o }))
        }

        function test_orientation_buttons_rotate(data) {
            const t = make({ screen: "unknown" })
            mouseClick(child(t.view, "orientation-" + data.o))
            compare(t.backend.calls, ["rotateTo " + data.o])
        }

        function test_rotate_cw_ccw() {
            const t = make()
            mouseClick(child(t.view, "rotateCW"))
            mouseClick(child(t.view, "rotateCCW"))
            compare(t.backend.calls, ["rotateBy 1", "rotateBy -1"])
        }

        function test_lock_switch() {
            const t = make({ locked: false })
            const sw = child(t.view, "lockSwitch")
            verify(!sw.checked)
            mouseClick(sw)
            compare(t.backend.calls, ["setLocked true"])
            t.backend.locked = true
            verify(sw.checked)
        }

        function test_mode_radios() {
            const t = make({ mode: "auto" })
            verify(child(t.view, "modeAuto").checked)
            verify(!child(t.view, "modeManual").checked)
            mouseClick(child(t.view, "modeManual"))
            compare(t.backend.calls, ["setMode manual"])
            t.backend.mode = "manual"
            verify(child(t.view, "modeManual").checked)
            verify(!child(t.view, "modeAuto").checked)
        }

        function test_no_imu_disables_modes_but_not_buttons() {
            const t = make({ imu: false })
            verify(child(t.view, "noSensor").visible)
            verify(!child(t.view, "modeAuto").enabled)
            verify(!child(t.view, "modeManual").enabled)
            verify(child(t.view, "rotateCW").enabled)
            verify(child(t.view, "lockSwitch").enabled)
        }

        function test_suggestion_banner() {
            const t = make({ suggested: "" })
            verify(!child(t.view, "suggestion").visible)
            t.backend.suggested = "portrait-cw"
            verify(child(t.view, "suggestion").visible)
            mouseClick(child(t.view, "acceptSuggestion"))
            compare(t.backend.calls, ["accept"])
        }

        function test_device_orientation_shown() {
            const t = make({ device: "portrait-cw" })
            verify(child(t.view, "deviceLabel").text.indexOf("Portrait (right grip down)") >= 0)
        }
    
        function test_settle_spin_shows_backend_value() {
            const t = make({ settleMs: 1500 })
            const spin = child(t.view, "settleSpin")
            compare(spin.value, 1500)
            compare(spin.displayText, "1.5 s")
            compare(spin.from, 100)
            compare(spin.to, 3000)
            t.backend.settleMs = 700
            compare(spin.value, 700)
        }

        function test_settle_spin_sets_backend() {
            const t = make({ settleMs: 500 })
            const spin = child(t.view, "settleSpin")
            spin.forceActiveFocus()
            keyClick(Qt.Key_Up)
            compare(t.backend.calls, ["setSettleMs 600"])
            // The daemon is the source of truth: the spin snaps back until it replies.
            compare(spin.value, 500)
            t.backend.settleMs = 600
            compare(spin.value, 600)
        }

        function test_prompt_spin_sets_backend() {
            const t = make({ mode: "manual", promptTimeoutMs: 5000 })
            const spin = child(t.view, "promptSpin")
            compare(spin.displayText, "5 s")
            spin.forceActiveFocus()
            keyClick(Qt.Key_Down)
            compare(t.backend.calls, ["setPromptTimeoutMs 4000"])
        }

        function test_prompt_spin_only_in_manual_mode() {
            const t = make({ mode: "auto" })
            verify(!child(t.view, "promptSpin").enabled)
            t.backend.mode = "manual"
            verify(child(t.view, "promptSpin").enabled)
        }

        function test_timing_disabled_without_sensor() {
            const t = make({ imu: false, mode: "manual" })
            verify(!child(t.view, "settleSpin").enabled)
            verify(!child(t.view, "promptSpin").enabled)
        }
    }
}
