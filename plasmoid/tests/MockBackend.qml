import QtQuick

// Stands in for DBusBackend; records every call.
QtObject {
    property bool available: true
    property bool imu: true
    property string mode: "auto"
    property bool locked: false
    property string device: "landscape"
    property string screen: "landscape"
    property string suggested: ""

    property var calls: []

    function record(c) {
        calls = calls.concat([c])
    }

    function setMode(m) { record("setMode " + m) }
    function setLocked(l) { record("setLocked " + l) }
    function rotateTo(o) { record("rotateTo " + o) }
    function rotateBy(n) { record("rotateBy " + n) }
    function accept() { record("accept") }
}
