package vm

import (
	"errors"
	"fmt"
	"log/slog"
)

const (
	MemorySize    = 4096
	StackSize     = 16
	RegisterCount = 16
	ScreenWidth   = 64
	ScreenHeight  = 32
	KeyCount      = 16

	ProgramStart    = uint16(0x200)
	InstructionSize = 2
)

type VM struct {
	memory    []uint8 // Memory (4k)
	registers []uint8 // V registers (V0-VF)

	stack []uint16 // Stack
	sp    uint16   // Stack pointer

	pc    uint16 // Program counter
	index uint16 // Index register

	delayTimer uint8 // Delay timer
	soundTimer uint8 // Sound timer

	gfx      []uint8 // Graphics buffer
	keypad   []uint8 // Keypad
	drawFlag bool    // Indicates a draw has occurred

	program []byte
}

func New(program []byte) *VM {
	return &VM{
		memory:    make([]uint8, MemorySize),
		registers: make([]uint8, RegisterCount),
		stack:     make([]uint16, StackSize),
		gfx:       make([]uint8, ScreenWidth*ScreenHeight),
		keypad:    make([]uint8, KeyCount),
		program:   program,
	}
}

type HAL interface {
	ReadInput(keyDown func(Key), keyUp func(Key)) error
	Draw(gfx []byte) error
	Beep() error
	WaitForNextFrame() error
	WaitForQuit() error
}

type Key uint8

const (
	Key0 = Key(iota)
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9
	KeyA
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
)

func (vm *VM) Run(hal HAL) error {
	vm.initialize()

	for {
		err := vm.runStep(hal)
		if err != nil {
			if errors.Is(err, errInfiniteLoop) {
				slog.Info("program looped")
				return vm.waitForReboot(hal)
			}

			return err
		}
	}
}

func (vm *VM) waitForReboot(hal HAL) error {
	for {
		if err := hal.WaitForNextFrame(); err != nil {
			return err
		}

		if err := hal.ReadInput(func(_ Key) {}, func(_ Key) {}); err != nil {
			return err
		}
	}
}

func (vm *VM) runStep(hal HAL) error {
	if err := vm.step(hal); err != nil {
		return err
	}

	if vm.drawFlag {
		if err := hal.Draw(vm.gfx); err != nil {
			return err
		}
		vm.drawFlag = false
	}

	if err := hal.ReadInput(vm.keyDown, vm.keyUp); err != nil {
		return err
	}

	if err := hal.WaitForNextFrame(); err != nil {
		return err
	}

	return nil
}

func (vm *VM) initialize() {
	vm.pc = ProgramStart
	vm.index = 0
	vm.sp = 0

	// Clear the display
	for i := range vm.gfx {
		vm.gfx[i] = 0
	}
	vm.drawFlag = true

	// Clear the stack, keypad, and V registers
	slog.Debug("clear stack", "n", len(vm.stack))
	for i := range vm.stack {
		vm.stack[i] = 0
	}

	slog.Debug("clear keypad", "n", len(vm.keypad))
	for i := range vm.keypad {
		vm.keypad[i] = 0
	}

	slog.Debug("clear registers", "n", len(vm.registers))
	for i := range vm.registers {
		vm.registers[i] = 0
	}

	// Clear memory
	slog.Debug("clear memory", "n", len(vm.memory))
	for i := range vm.memory {
		vm.memory[i] = 0
	}

	// Load font set into memory
	slog.Debug("load font", "at", fmt.Sprintf("0x%04x", 0), "n", len(chip8Font))
	copy(vm.memory[0:], chip8Font)

	// Load program into memory
	slog.Info("load program", "at", fmt.Sprintf("0x%04x", ProgramStart), "n", len(vm.program))
	copy(vm.memory[ProgramStart:], vm.program)

	// Reset timers
	vm.delayTimer = 0
	vm.soundTimer = 0
}

func (vm *VM) keyDown(key Key) {
	vm.keypad[int(key)] = 1
}

func (vm *VM) keyUp(key Key) {
	vm.keypad[int(key)] = 0
}

func (vm *VM) step(hal HAL) error {
	if err := vm.executeOpcode(vm.fetchOpcode()); err != nil {
		return err
	}

	// Update timers
	if vm.delayTimer > 0 {
		vm.delayTimer--
	}

	if vm.soundTimer > 0 {
		if vm.soundTimer == 1 {
			if err := hal.Beep(); err != nil {
				return err
			}
		}
		vm.soundTimer--
	}

	return nil
}

func (vm *VM) fetchOpcode() uint16 {
	hi := vm.memory[vm.pc]
	lo := vm.memory[vm.pc+1]

	opcode := uint16(hi)<<8 | uint16(lo) // Op code is two bytes
	return opcode
}
