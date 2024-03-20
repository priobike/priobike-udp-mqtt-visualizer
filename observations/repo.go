package observations

import "sync"

// The current primary signal for a thing.
var currentPrimarySignal = &sync.Map{}

// Get the most recent primary signal for this thing.
func GetCurrentPrimarySignal(thingName string) (*Observation, bool) {
	signal, ok := currentPrimarySignal.Load(thingName)
	if !ok {
		return &Observation{}, false
	}
	return signal.(*Observation), true
}

// The current signal program for a thing.
var currentSignalProgram = &sync.Map{}

// Get the most recent signal program for this thing.
func GetCurrentSignalProgram(thingName string) (*Observation, bool) {
	signal, ok := currentSignalProgram.Load(thingName)
	if !ok {
		return &Observation{}, false
	}
	return signal.(*Observation), true
}

// The current car detector for a thing.
var currentCarDetector = &sync.Map{}

// Get the most recent car detector for this thing.
func GetCurrentCarDetector(thingName string) (*Observation, bool) {
	signal, ok := currentCarDetector.Load(thingName)
	if !ok {
		return &Observation{}, false
	}
	return signal.(*Observation), true
}

// The current bike detector for a thing.
var currentBikeDetector = &sync.Map{}

// Get the most recent bike detector for this thing.
func GetCurrentBikeDetector(thingName string) (*Observation, bool) {
	signal, ok := currentBikeDetector.Load(thingName)
	if !ok {
		return &Observation{}, false
	}
	return signal.(*Observation), true
}

// The current cycle second for a thing.
var currentCycleSecond = &sync.Map{}

// Get the most recent cycle second for this thing.
func GetCurrentCycleSecond(thingName string) (*Observation, bool) {
	signal, ok := currentCycleSecond.Load(thingName)
	if !ok {
		return &Observation{}, false
	}
	return signal.(*Observation), true
}
