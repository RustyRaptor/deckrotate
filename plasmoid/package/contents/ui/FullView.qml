import QtQuick
import QtQuick.Layouts
import org.kde.kirigami as Kirigami
import org.kde.plasma.components as PlasmaComponents3
import org.kde.plasma.extras as PlasmaExtras
import "logic.js" as Logic

Item {
    id: view

    // DBusBackend in the applet, MockBackend in tests.
    property var backend

    implicitWidth: Kirigami.Units.gridUnit * 18
    implicitHeight: content.implicitHeight + Kirigami.Units.largeSpacing * 2

    PlasmaExtras.PlaceholderMessage {
        objectName: "notRunning"
        anchors.centerIn: parent
        width: parent.width - Kirigami.Units.largeSpacing * 4
        visible: !view.backend.available
        iconName: "transform-rotate"
        text: tr("Rotation service is not running")
        explanation: tr("Start it with: systemctl --user start deckrotate")
    }

    ColumnLayout {
        id: content
        objectName: "controls"
        visible: view.backend.available
        anchors.fill: parent
        anchors.margins: Kirigami.Units.largeSpacing
        spacing: Kirigami.Units.smallSpacing

        Kirigami.InlineMessage {
            objectName: "suggestion"
            Layout.fillWidth: true
            visible: view.backend.suggested !== ""
            type: Kirigami.MessageType.Information
            text: tr("The Deck is turned to %1", Logic.label(view.backend.suggested))
            actions: [
                Kirigami.Action {
                    id: acceptAction
                    text: tr("Rotate")
                    icon.name: "object-rotate-right"
                    onTriggered: view.backend.accept()
                }
            ]
        }
        // Mirrors acceptAction as a plain button tests and keyboards can reach.
        PlasmaComponents3.Button {
            objectName: "acceptSuggestion"
            visible: view.backend.suggested !== ""
            Layout.alignment: Qt.AlignRight
            action: acceptAction
        }

        Kirigami.InlineMessage {
            objectName: "noSensor"
            Layout.fillWidth: true
            visible: !view.backend.imu
            type: Kirigami.MessageType.Warning
            text: tr("Orientation sensor unavailable; rotate manually.")
        }

        PlasmaExtras.Heading {
            level: 5
            text: tr("Screen orientation")
        }

        GridLayout {
            Layout.fillWidth: true
            columns: 2
            columnSpacing: Kirigami.Units.smallSpacing
            rowSpacing: Kirigami.Units.smallSpacing

            Repeater {
                model: Logic.orientations
                delegate: PlasmaComponents3.ToolButton {
                    required property string modelData
                    objectName: "orientation-" + modelData
                    Layout.fillWidth: true
                    checkable: true
                    checked: view.backend.screen === modelData
                    // Leave `checked` to the binding; the daemon's reply updates it.
                    onClicked: {
                        checked = Qt.binding(() => view.backend.screen === modelData)
                        view.backend.rotateTo(modelData)
                    }
                    contentItem: RowLayout {
                        spacing: Kirigami.Units.smallSpacing
                        OrientationGlyph {
                            angle: Logic.angle(modelData)
                        }
                        PlasmaComponents3.Label {
                            Layout.fillWidth: true
                            text: Logic.label(modelData)
                            wrapMode: Text.Wrap
                        }
                    }
                    Accessible.name: Logic.label(modelData)
                }
            }
        }

        RowLayout {
            Layout.fillWidth: true
            PlasmaComponents3.Button {
                objectName: "rotateCCW"
                Layout.fillWidth: true
                icon.name: "object-rotate-left"
                text: tr("Rotate left")
                onClicked: view.backend.rotateBy(-1)
            }
            PlasmaComponents3.Button {
                objectName: "rotateCW"
                Layout.fillWidth: true
                icon.name: "object-rotate-right"
                text: tr("Rotate right")
                onClicked: view.backend.rotateBy(1)
            }
        }

        Kirigami.Separator {
            Layout.fillWidth: true
            Layout.topMargin: Kirigami.Units.smallSpacing
            Layout.bottomMargin: Kirigami.Units.smallSpacing
        }

        PlasmaComponents3.Switch {
            objectName: "lockSwitch"
            Layout.fillWidth: true
            text: tr("Lock rotation")
            checked: view.backend.locked
            onToggled: {
                const want = checked
                checked = Qt.binding(() => view.backend.locked)
                view.backend.setLocked(want)
            }
        }

        PlasmaExtras.Heading {
            level: 5
            text: tr("When the Deck is turned")
        }

        PlasmaComponents3.RadioButton {
            objectName: "modeAuto"
            enabled: view.backend.imu
            text: tr("Rotate automatically")
            checked: view.backend.mode === "auto"
            onClicked: {
                checked = Qt.binding(() => view.backend.mode === "auto")
                view.backend.setMode("auto")
            }
        }
        PlasmaComponents3.RadioButton {
            objectName: "modeManual"
            enabled: view.backend.imu
            text: tr("Show a rotate button")
            checked: view.backend.mode === "manual"
            onClicked: {
                checked = Qt.binding(() => view.backend.mode === "manual")
                view.backend.setMode("manual")
            }
        }

        GridLayout {
            Layout.fillWidth: true
            Layout.topMargin: Kirigami.Units.smallSpacing
            columns: 2

            PlasmaComponents3.Label {
                Layout.fillWidth: true
                text: tr("Rotate after holding still for")
                wrapMode: Text.Wrap
            }
            SecondsSpinBox {
                objectName: "settleSpin"
                Layout.preferredWidth: Kirigami.Units.gridUnit * 6
                enabled: view.backend.imu
                range: Logic.settleRange
                backendValue: view.backend.settleMs
                onChosen: (ms) => view.backend.setSettleMs(ms)
            }

            PlasmaComponents3.Label {
                Layout.fillWidth: true
                text: tr("Keep the rotate button for")
                wrapMode: Text.Wrap
            }
            SecondsSpinBox {
                objectName: "promptSpin"
                Layout.preferredWidth: Kirigami.Units.gridUnit * 6
                enabled: view.backend.imu && view.backend.mode === "manual"
                range: Logic.promptRange
                backendValue: view.backend.promptTimeoutMs
                onChosen: (ms) => view.backend.setPromptTimeoutMs(ms)
            }
        }

        PlasmaComponents3.Label {
            objectName: "deviceLabel"
            Layout.fillWidth: true
            Layout.topMargin: Kirigami.Units.smallSpacing
            opacity: 0.7
            font: Kirigami.Theme.smallFont
            wrapMode: Text.Wrap
            text: tr("Deck is held: %1", Logic.label(view.backend.device))
        }

        Item {
            Layout.fillHeight: true
        }
    }

    function tr(s, arg) {
        if (typeof i18nd === "function")
            return arg === undefined ? i18nd(Logic.domain, s) : i18nd(Logic.domain, s, arg)
        return arg === undefined ? s : s.replace("%1", arg)
    }
}
