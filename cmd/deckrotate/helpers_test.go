package main

import "github.com/RustyRaptor/deckrotate/internal/service"

var statusFixture = service.Status{
	Mode: "manual", Locked: true, Device: "portrait-cw",
	Screen: "landscape", Suggested: "portrait-cw", IMU: true,
}
