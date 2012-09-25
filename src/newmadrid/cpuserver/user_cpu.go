package main

import (
	"newmadrid/msp43x"
)

func newMemory() *msp43x.HookableMemory {
	m := msp43x.HookableMemory{}
	
	return &m
}

func NewUserCpu() *UserCpu {
	return &UserCpu{
		MCU: &msp43x.CPU{},
		Mem: newMemory(),
		Image: "boot",
		State: CpuStopped,
	}
}

const (
	CpuStopped	= iota
	CpuRunning
	CpuFault
	CpuStepping
)

type UserCpu struct {
	MCU *msp43x.CPU
	Mem *msp43x.HookableMemory

	Image string

	State int
}

func (ucpu *UserCpu) Run() {

}
