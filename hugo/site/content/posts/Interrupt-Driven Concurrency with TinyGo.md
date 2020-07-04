---
title: "Interrupt-Driven Concurrency With TinyGo"
date: 2020-07-04T09:10:46-04:00
draft: false
---

Interrupts are one of the most important mechanisms in microcontrollers.
An "interrupt" is essentially a callback triggered by hardware to allow a CPU to process an event.
The CPU interrupts whatever it was previously working on, saves the current processing state, and invokes a function.

In TinyGo v0.14, channels have been adapted to allow interrupts to interact with goroutines.
This allows for goroutines to effectively process hardware events using mechanisms already included in the language.

Now it is possible to do non-blocking selects from within an interrupt, with the same semantics as non-blocking selects have everywhere else in Go.
If an interrupt unblocks a goroutine that was waiting on a channel, this wakes up the scheduler and resumes the goroutine.

# Morse Code with a Timer

Timer interrupts are one of the easiest kinds of interrupts to work with.
We are going to use the "systick" interrupt included on many ARM CPU cores as a timer, and use this to blink morse code with an LED.

_NOTE: all included code snippets have been tested tested on an [Arduino Nano 33 IoT](https://store.arduino.cc/usa/nano-33-iot) and an [Adafruit Metro M4 Express AirLift](https://www.adafruit.com/product/4000)_

## Morse Code

Morse code breaks text down into 4 different symbols:
- dot
- dash
- inter-letter space
- inter-word space

Morse code encodes data by using lengths of the presence and absence of a signal, measured in multiples of a time unit.
Here we will use a 1 second time unit.
Each letter is broken down into a series of dots and dashes.
A dot is a signal lasting for a single time unit, and a dash is a signal lasting 3 time units.
Dots and dashes are seperated by 1 time unit spaces.
The inter-letter space is a pause lasting for 3 units, and an inter-word space is a pause lasting for 7 units.

It is convenient to represent the primitive symbols as signed integers - we can represent signal lengths with positive integers and pause lengths with negative integers:

```Go
type symbol int8

const (
	dot         symbol = 1
	dash        symbol = 3
	letterSpace symbol = -1 // excludes inter-symbol spaces
)
```

Now we can use a map to easily define characters in terms of these symbols, using a constant map:
```Go
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
```

## Planning Concurrency

Now we need to figure out how the interrupt should communicate with a user goroutine sending morse data.
We need 2 mechanisms:
1. The user needs to be able to send morse data.
2. The interrupt needs to notify the goroutine once all data has been sent.

We can do this with a pair of channels:
- a (preferably) buffered channel of morse symbols
- an unbuffered channel for notifying completion

```Go
var send = make(chan symbol, 16)
var done = make(chan struct{})
```

## Writing an Interrupt Handler

### Setup

We are going to need to pull in a few packages:
```Go
import (
	"device/arm"
	"machine"
)
```

The `device/arm` package includes bindings to some CPU controls (systick and the interrupt controller).
The `machine` acts as a hardware abstraction layer for microcontroller boards, providing pin and perhipheral bindings.

First, we need to bind an interrupt handler to systick:
```Go
//export SysTick_Handler
func timer_isr() {
    // handle the interrupt
}
```

_NOTE: TinyGo has a `runtime/interrupt` package which provides a cleaner interface, but it does not currently work with systick_

The `device/arm` includes a function to set up the systick timer:
```Go
arm.SetupSystemTimer(machine.CPUFrequency()/10)
```

This code gets the CPU's frequency, and sets the interval between timer interrupts to 1/10 of the cycles in 1 second.
Therefore, our timer interrupt should be invoked once every 100ms.

### Handling the Interrupt

There are a couple things we need to keep in mind when writing an interrupt handler:
1. The goroutine sending us information could potentially fall behind.
2. An interrupt cannot block since it is not in a goroutine. This means that we cannot use any blocking channel operations like a direct send or recieve.

First, we need to keep track of the current state of the current symbol:
```Go
var state symbol
```

We can use this to represent the portion of the current symbol which still needs to be processed.
When this is positive we need the LED output to be high in order to signal.
When this is negative we need the LED output to be low to pause.
We want this to move towards 0 as the interval progresses so that we know when we are done.
```Go
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
}
```

We have a few scenarios to think about for getting the next symbol:
1. Another symbol was sent for us to process on the `send` channel.
2. The sender has finished, and is waiting for a signal that the transmission is done.
3. The sender is not currently sending, and there is no symbol to process.
4. We do not want to spend very long in the interrupt handler - it is blocking other things from running.

This maps fairly simply to a select statement:
```Go
select {
case state = <-send:
    // We have a new symbol to send.
case done <- struct{}{}:
    // Signal completion.
    // The channel is unbuffered, so this is a no-op if nothing is waiting for a completion notification.
default:
    // No more symbols for now.
}
```

Altogether, our interrupt handler is:
```Go
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
```

## Sending Data to the Interrupt

Now all we have to do is convert text to morse and send it:
```Go
for _, c := range str {
    for _, s := range morseTable[c] {
        send <- s
    }
    send <- letterSpace
}
```

_NOTE: if the buffer in our send channel fills, the sending goroutine will wait until a symbol is processed before continuing_

Then wait for completion:
```Go
defer func() { <-done }()
```

We can wrap this up in a simple function:
```Go
func signalMorse(str string) {
    defer func() { <-done }()

    for _, c := range str {
        for _, s := range morseTable[c] {
            send <- s
        }
        send <- letterSpace
    }
}
```

## Wrapping it up in main
Now all we need is a main function to tie everything together:
```Go
func main() {
    machine.LED.Configure(machine.PinConfig{Mode: machine.PinOutput})

	arm.SetupSystemTimer(machine.CPUFrequency() / 10)

	for {
		signalMorse("SOS ")
	}
}
```

The complete code from this example is available [here](/interrupt.go).