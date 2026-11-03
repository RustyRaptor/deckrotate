import QtQuick
import QtQuick.Layouts
import org.kde.plasma.plasmoid
import org.kde.plasma.core as PlasmaCore
import org.kde.kirigami as Kirigami
import "logic.js" as Logic

PlasmoidItem {
    id: root

    readonly property DBusBackend backend: DBusBackend {}

    Plasmoid.icon: Logic.iconName(backend)
    Plasmoid.status: PlasmaCore.Types.ActiveStatus
    toolTipMainText: i18n("Screen Rotation")
    toolTipSubText: Logic.summary(backend)

    // Left click opens the panel, middle click toggles the lock.
    compactRepresentation: MouseArea {
        acceptedButtons: Qt.LeftButton | Qt.MiddleButton
        hoverEnabled: true
        onClicked: (mouse) => {
            if (mouse.button === Qt.MiddleButton)
                root.backend.setLocked(!root.backend.locked)
            else
                root.expanded = !root.expanded
        }

        Kirigami.Icon {
            anchors.fill: parent
            source: Plasmoid.icon
            active: parent.containsMouse
        }
    }

    fullRepresentation: FullView {
        backend: root.backend
        Layout.minimumWidth: Kirigami.Units.gridUnit * 16
        Layout.minimumHeight: implicitHeight
        Layout.preferredWidth: implicitWidth
        Layout.preferredHeight: implicitHeight
    }

    Plasmoid.contextualActions: [
        PlasmaCore.Action {
            text: i18n("Lock Rotation")
            icon.name: "object-locked"
            checkable: true
            checked: root.backend.locked
            enabled: root.backend.available
            onTriggered: root.backend.setLocked(checked)
        },
        PlasmaCore.Action {
            text: i18n("Rotate Right")
            icon.name: "object-rotate-right"
            enabled: root.backend.available
            onTriggered: root.backend.rotateBy(1)
        },
        PlasmaCore.Action {
            text: i18n("Rotate Left")
            icon.name: "object-rotate-left"
            enabled: root.backend.available
            onTriggered: root.backend.rotateBy(-1)
        }
    ]
}
