import QtQuick
import org.kde.kirigami as Kirigami

// A tiny Deck silhouette turned to `angle`; the bar marks the Deck's top edge.
Item {
    id: glyph

    property int angle: 0
    property color color: Kirigami.Theme.textColor

    implicitWidth: Kirigami.Units.iconSizes.medium
    implicitHeight: Kirigami.Units.iconSizes.medium

    Item {
        anchors.centerIn: parent
        width: glyph.width * 0.9
        height: width * 0.5
        rotation: glyph.angle

        Rectangle {
            anchors.fill: parent
            radius: height * 0.25
            color: "transparent"
            border.color: glyph.color
            border.width: 1.5
        }
        // Screen.
        Rectangle {
            anchors.centerIn: parent
            width: parent.width * 0.5
            height: parent.height * 0.6
            color: Qt.alpha(glyph.color, 0.35)
        }
        // Top edge marker.
        Rectangle {
            anchors.horizontalCenter: parent.horizontalCenter
            anchors.top: parent.top
            anchors.topMargin: -3
            width: parent.width * 0.5
            height: 2
            radius: 1
            color: Kirigami.Theme.highlightColor
        }
    }
}
