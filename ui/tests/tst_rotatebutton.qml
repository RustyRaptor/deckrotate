import QtQuick
import QtTest
import "../components"

Item {
    id: root
    width: 200
    height: 200

    Component {
        id: buttonComponent
        RotateButton {}
    }

    SignalSpy {
        id: spy
        signalName: "accepted"
    }

    TestCase {
        name: "RotateButton"
        when: windowShown

        function init() {
            spy.clear()
        }

        function test_icon_follows_angle_data() {
            return [0, 90, 180, 270].map(a => ({ tag: String(a), angle: a }))
        }

        function test_icon_follows_angle(data) {
            const b = createTemporaryObject(buttonComponent, root, { angle: data.angle })
            compare(b.iconRotation, data.angle)
        }

        function test_tap_accepts() {
            const b = createTemporaryObject(buttonComponent, root)
            spy.target = b
            mouseClick(b)
            compare(spy.count, 1)
        }

        function test_touch_accepts() {
            const b = createTemporaryObject(buttonComponent, root)
            spy.target = b
            const touch = touchEvent(b)
            touch.press(0, b, b.width / 2, b.height / 2).commit()
            touch.release(0, b, b.width / 2, b.height / 2).commit()
            compare(spy.count, 1)
        }

        function test_tap_outside_circle_ignored() {
            const b = createTemporaryObject(buttonComponent, root)
            spy.target = b
            mouseClick(b, 1, 1) // square corner, outside the round button
            compare(spy.count, 0)
        }

        function test_is_finger_sized() {
            const b = createTemporaryObject(buttonComponent, root)
            verify(b.width >= 56 && b.height >= 56, "button should be at least 56px for touch")
        }

        function test_accessible() {
            const b = createTemporaryObject(buttonComponent, root)
            compare(b.Accessible.role, Accessible.Button)
            verify(b.Accessible.name.length > 0)
        }
    }
}
