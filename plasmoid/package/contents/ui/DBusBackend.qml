import QtQuick
import org.kde.plasma.workspace.dbus as DBus

// The deckrotate daemon as seen over the session bus. Same API as MockBackend.
QtObject {
    id: backend

    readonly property string service: "io.github.RustyRaptor.DeckRotate"
    readonly property string objectPath: "/io/github/RustyRaptor/DeckRotate"

    readonly property bool available: watcher.registered
    readonly property bool imu: available && props.properties.ImuAvailable === true
    readonly property string mode: text(props.properties.Mode, "auto")
    readonly property bool locked: available && props.properties.Locked === true
    readonly property string device: text(props.properties.DeviceOrientation, "unknown")
    readonly property string screen: text(props.properties.ScreenOrientation, "unknown")
    readonly property string suggested: text(props.properties.SuggestedOrientation, "")
    readonly property int settleMs: number(props.properties.SettleMs, 500)
    readonly property int promptTimeoutMs: number(props.properties.PromptTimeoutMs, 5000)

    property DBus.DBusServiceWatcher watcher: DBus.DBusServiceWatcher {
        busType: DBus.BusType.Session
        watchedService: backend.service
        // The daemon may start after the applet, or restart.
        onRegisteredChanged: if (registered) backend.props.updateAll()
    }

    property DBus.Properties props: DBus.Properties {
        busType: DBus.BusType.Session
        service: backend.service
        path: backend.objectPath
        iface: backend.service
    }

    // Strings arrive as String objects; make them plain for === comparisons.
    function text(v, fallback) {
        return available && v !== undefined && v !== null ? String(v) : fallback
    }

    function number(v, fallback) {
        return available && v !== undefined && v !== null ? Number(v) : fallback
    }

    // No signature: the module rejects explicit ones but infers types correctly.
    function call(member, args) {
        const reply = DBus.SessionBus.asyncCall({
            service: service,
            path: objectPath,
            iface: service,
            member: member,
            arguments: args
        })
        reply.finished.connect(() => {
            if (reply.isError)
                console.warn("deckrotate:", member, "failed:", reply.error.message)
        })
    }

    function setMode(m) { call("SetMode", [m]) }
    function setLocked(l) { call("SetLocked", [l]) }
    function rotateTo(o) { call("RotateTo", [o]) }
    function rotateBy(n) { call("RotateBy", [new DBus.int32(n)]) }
    function setSettleMs(ms) { call("SetSettleMs", [new DBus.int32(ms)]) }
    function setPromptTimeoutMs(ms) { call("SetPromptTimeoutMs", [new DBus.int32(ms)]) }
    function accept() { call("Accept", []) }
}
