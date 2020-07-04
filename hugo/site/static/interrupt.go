package main

import (
	"device/arm"
	"machine"
)

type symbol int8

const (
	dot         symbol = 1
	dash        symbol = 3
	letterSpace symbol = -1 // excludes inter-symbol spaces
)

var morseTable = map[rune][]symbol{
	'A': {dot, dash},
	'B': {dash, dot, dot, dot},
	'C': {dash, dot, dash, dot},
	'D': {dash, dot, dot},
	'E': {dot},
	'F': {dot, dot, dash, dot},
	'G': {dash, dash, dot},
	'H': {dot, dot, dot, dot},
	'I': {dot, dot},
	'J': {dot, dash, dash, dash},
	'K': {dash, dot, dash},
	'L': {dot, dash, dot, dot},
	'M': {dash, dash},
	'N': {dash, dot},
	'O': {dash, dash, dash},
	'P': {dot, dash, dash, dot},
	'Q': {dash, dash, dot, dash},
	'R': {dot, dash, dot},
	'S': {dot, dot, dot},
	'T': {dash},
	'U': {dot, dot, dash},
	'V': {dot, dot, dot, dash},
	'W': {dot, dash, dash},
	'X': {dash, dot, dot, dash},
	'Y': {dash, dot, dash, dash},
	'Z': {dash, dash, dot, dot},
	'1': {dot, dash, dash, dash, dash},
	'2': {dot, dot, dash, dash, dash},
	'3': {dot, dot, dot, dash, dash},
	'4': {dot, dot, dot, dot, dash},
	'5': {dot, dot, dot, dot, dot},
	'6': {dash, dot, dot, dot, dot},
	'7': {dash, dash, dot, dot, dot},
	'8': {dash, dash, dash, dot, dot},
	'9': {dash, dash, dash, dash, dot},
	'0': {dash, dash, dash, dash, dash},
	' ': {3 - 7}, // leading symbol pause + following symbol pause + inter-letter space = 3
}

var send = make(chan symbol, 16)
var done = make(chan struct{})

var state symbol

func signalMorse(str string) {
	defer func() { <-done }()

	for _, c := range str {
		for _, s := range morseTable[c] {
			send <- s
		}
		send <- letterSpace
	}
}

//export SysTick_Handler
func timer_isr() {
	switch {
	case state > 0:
		machine.LED.High()
		state--
	case state < 0:
		machine.LED.Low()
		state++
	default:
		machine.LED.Low()

		// We need to fetch another symbol.
		// This also counts as our inter-symbol pause.
		select {
		case state = <-send:
			// We have a new symbol to send.
		case done <- struct{}{}:
			// Signal completion.
			// The channel is unbuffered, so this is a no-op if nothing is waiting for a completion notification.
		default:
			// No more symbols for now.
		}
	}
}

func main() {
	machine.LED.Configure(machine.PinConfig{Mode: machine.PinOutput})

	arm.SetupSystemTimer(machine.CPUFrequency() / 10)

	for {
		signalMorse("SOS ")
	}
}
