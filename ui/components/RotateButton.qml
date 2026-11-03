import QtQuick
import org.kde.kirigami as Kirigami

// Round "rotate to match the Deck" button for the manual-mode overlay.
Item {
    id: button

    property int angle: 0
    readonly property alias iconRotation: icon.rotation
    signal accepted

    implicitWidth: 64
    implicitHeight: 64
    width: implicitWidth
    height: implicitHeight

    Accessible.role: Accessible.Button
    Accessible.name: qsTr("Rotate screen to match the Deck")
    Accessible.onPressAction: button.accepted()

    Rectangle {
        id: disc
        anchors.fill: parent
        radius: width / 2
        color: Qt.alpha(Kirigami.Theme.backgroundColor, tap.pressed ? 0.95 : 0.85)
        border.color: Kirigami.Theme.highlightColor
        border.width: 2
        scale: tap.pressed ? 0.92 : 1

        Behavior on scale {
            NumberAnimation { duration: 80 }
        }

        Kirigami.Icon {
            id: icon
            anchors.centerIn: parent
            width: parent.width * 0.55
            height: width
            source: "object-rotate-right"
            rotation: button.angle
        }
    }

    TapHandler {
        id: tap
        // Only taps on the disc itself, not the square's corners.
        onTapped: (point) => {
            const dx = point.position.x - button.width / 2
            const dy = point.position.y - button.height / 2
            if (dx * dx + dy * dy <= (button.width / 2) * (button.width / 2))
                button.accepted()
        }
    }

    // Pop in so the button catches the eye after the Deck is turned.
    NumberAnimation on opacity {
        from: 0
        to: 1
        duration: 150
        running: true
    }
}
