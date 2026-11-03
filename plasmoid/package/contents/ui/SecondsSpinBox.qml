import QtQuick
import org.kde.plasma.components as PlasmaComponents3
import "logic.js" as Logic

// Milliseconds shown as seconds. Follows backendValue; edits are reported
// through chosen() and the box snaps back until the daemon confirms.
PlasmaComponents3.SpinBox {
    id: spin

    property int backendValue
    property var range
    signal chosen(int ms)

    from: range.from
    to: range.to
    stepSize: range.step
    editable: true
    value: backendValue

    textFromValue: (v, locale) => Logic.seconds(v)
    valueFromText: (text, locale) => {
        const ms = Logic.parseSeconds(text)
        return isNaN(ms) ? spin.value : ms
    }
    validator: RegularExpressionValidator {
        regularExpression: /[0-9]+([.,][0-9]+)?\s*s?/
    }

    onValueModified: {
        const ms = value
        value = Qt.binding(() => spin.backendValue)
        spin.chosen(ms)
    }
}
