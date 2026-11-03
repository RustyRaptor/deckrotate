import QtQuick
import org.kde.layershell as LayerShell
import "components"
import "js/args.js" as Args

// Manual-mode prompt. Usage:
//   qml overlay.qml -- --vertical top|bottom --horizontal left|right --angle 0|90|180|270
// Exits with code 10 when tapped; the daemon kills it on timeout.
Window {
    id: window

    readonly property var opts: Args.parse(Qt.application.arguments)
    readonly property var edges: Args.edges(opts)
    readonly property int inset: 20

    screen: Args.panelScreen(Qt.application.screens)
    width: button.width + inset
    height: button.height + inset
    visible: true
    color: "transparent"
    flags: Qt.FramelessWindowHint | Qt.WindowDoesNotAcceptFocus
    title: "deckrotate"

    LayerShell.Window.scope: "deckrotate"
    LayerShell.Window.layer: LayerShell.Window.LayerOverlay
    LayerShell.Window.anchors: (edges.top ? LayerShell.Window.AnchorTop : LayerShell.Window.AnchorBottom)
                             | (edges.left ? LayerShell.Window.AnchorLeft : LayerShell.Window.AnchorRight)
    LayerShell.Window.keyboardInteractivity: LayerShell.Window.KeyboardInteractivityNone
    LayerShell.Window.exclusionZone: -1

    RotateButton {
        id: button
        x: window.edges.left ? window.inset : 0
        y: window.edges.top ? window.inset : 0
        angle: window.opts.angle
        onAccepted: Qt.exit(10)
    }
}
