package vm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
)

var (
	errInfiniteLoop = errors.New("infinite loop")
)

func (vm *VM) executeOpcode(opcode uint16) error {
	instr := decode(opcode)

	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		slog.Debug(
			"exec",
			"pc", fmt.Sprintf("0x%04x", vm.pc),
			"opcode", fmt.Sprintf("0x%04x", opcode),
			"instr", instr.Name(opcode),
		)
	}

	return instr.Execute(vm, opcode)
}

type instruction struct {
	Name    func(opcode uint16) string
	Execute func(vm *VM, opcode uint16) error
}

func decode(opcode uint16) instruction {
	switch opcode & 0xF000 {
	case 0x0000:
		switch opcode & 0x00FF {
		case 0x00E0:
			// 00E0 - Clear screen
			return clsInstruction

		case 0x00EE:
			// 00EE - Return from subroutine
			return rtsInstruction
		}

	case 0x1000:
		// 1NNN - Jumps to address NNN
		return jmpInstruction

	case 0x2000:
		// 2NNN - Calls subroutine at NNN
		return jsrInstruction

	case 0x3000:
		// 3XNN - Skips the next instruction if VX equals NN
		return skeq1Instruction

	case 0x4000:
		// 4XNN - Skips the next instruction if VX does not equal NN
		return skne1Instruction

	case 0x5000:
		// 5XY0 - Skips the next instruction if VX equals VY
		return skeq2Instruction

	case 0x6000:
		// 6XNN - Sets VX to NN
		return mov1Instruction

	case 0x7000:
		// 7XNN - Adds NN to VX
		return add1Instruction

	case 0x8000:
		// 8XY_
		switch opcode & 0x000F {
		case 0x0000:
			// 8XY0 - Sets VX to the value of VY
			return mov2Instruction

		case 0x0001:
			// 8XY1 - Sets VX to (VX OR VY)
			return orInstruction

		case 0x0002:
			// 8XY2 - Sets VX to (VX AND VY)
			return andInstruction

		case 0x0003:
			// 8XY3 - Sets VX to (VX XOR VY)
			return xorInstruction

		case 0x0004:
			// 8XY4 - Adds VY to VX. VF is set to 1 when there's a carry, and to 0 when there isn't.
			return add2Instruction

		case 0x0005:
			// 8XY5 - VY is subtracted from VX. VF is set to 0 when there's a borrow, and 1 when there isn't.
			return subInstruction

		case 0x0006:
			// 0x8XY6 - Shifts VX right by one. VF is set to the value of the least significant bit of VX before the shift.
			return shrInstruction

		case 0x0007:
			// 0x8XY7: Sets VX to VY minus VX. VF is set to 0 when there's a borrow, and 1 when there isn't.
			return rsbInstruction

		case 0x000E:
			// 0x8XYE: Shifts VX left by one. VF is set to the value of the most significant bit of VX before the shift.
			return shlInstruction
		}

	case 0x9000:
		// 9XY0 - Skips the next instruction if VX doesn't equal VY
		return skne2Instruction

	case 0xA000:
		// ANNN - Sets I to the address NNN
		return mviInstruction

	case 0xB000:
		// BNNN - Jumps to the address NNN plus V0
		return jmiInstruction

	case 0xC000:
		// CXNN - Sets VX to a random number, masked by NN
		return randInstruction

	case 0xD000:
		// DXYN: Draws a sprite at coordinate (VX, VY) that has a width of 8
		// pixels and a height of N pixels.
		// Each row of 8 pixels is read as bit-coded starting from memory
		// location I;
		// I value doesn't change after the execution of this instruction.
		// VF is set to 1 if any screen pixels are flipped from set to unset
		// when the sprite is drawn, and to 0 if that doesn't happen.
		return spriteInstruction

	case 0xE000:
		switch opcode & 0x00FF {
		case 0x009E:
			// EX9E - Skips the next instruction if the key stored in VX is pressed
			return skprInstruction

		case 0x00A1:
			// EXA1 - Skips the next instruction if the key stored in VX isn't pressed
			return skupInstruction
		}

	case 0xF000:
		switch opcode & 0x00FF {
		case 0x0007:
			// FX07 - Sets VX to the value of the delay timer
			return gdelayInstruction

		case 0x000A:
			// FX0A - A key press is awaited, and then stored in VX
			return keyInstruction

		case 0x0015:
			// FX15 - Sets the delay timer to VX
			return sdelayInstruction

		case 0x0018:
			// FX18 - Sets the sound timer to VX
			return ssoundInstruction

		case 0x001E:
			// FX1E - Adds VX to I
			// VF is set to 1 when range overflow (I+VX>0xFFF), and 0
			// when there isn't.
			return adiInstruction

		case 0x0029:
			// FX29 - Sets I to the location of the sprite for the
			// character in VX. Characters 0-F (in hexadecimal) are
			// represented by a 4x5 font
			return fontInstruction

		case 0x0033:
			// FX33 - Stores the Binary-coded decimal representation of VX
			// at the addresses I, I plus 1, and I plus 2
			return bcdInstruction

		case 0x0055:
			// FX55 - Stores V0 to VX in memory starting at address I
			return strInstruction

		case 0x0065:
			// FX65 - Reads memory starting at address I into V0...VX
			return ldrInstruction
		}
	}

	return unknownInstruction
}

var (
	// 00E0	cls	Clear the screen
	clsInstruction = instruction{
		Name: func(opcode uint16) string {
			return "cls"
		},
		Execute: func(vm *VM, opcode uint16) error {
			for i := range vm.gfx {
				vm.gfx[i] = 0
			}
			vm.drawFlag = true
			vm.pc += InstructionSize
			return nil
		},
	}

	// 00EE	rts	return from subroutine call
	rtsInstruction = instruction{
		Name: func(opcode uint16) string {
			return "rts"
		},
		Execute: func(vm *VM, opcode uint16) error {
			vm.sp--
			vm.pc = vm.stack[vm.sp]
			vm.pc += InstructionSize
			return nil
		},
	}

	// 1xxx	jmp xxx	jump to address xxx
	jmpInstruction = instruction{
		Name: func(opcode uint16) string {
			return fmt.Sprintf("jmp 0x%04x", opcode&0x0FFF)
		},
		Execute: func(vm *VM, opcode uint16) error {
			pc := opcode & 0x0FFF
			if pc == vm.pc {
				return errInfiniteLoop
			}
			vm.pc = pc
			return nil
		},
	}

	// 2xxx	jsr xxx	jump to subroutine at address xxx
	jsrInstruction = instruction{
		Name: func(opcode uint16) string {
			return fmt.Sprintf("jsr 0x%04x", opcode&0x0FFF)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vm.stack[vm.sp] = vm.pc
			vm.sp++
			vm.pc = opcode & 0x0FFF
			return nil
		},
	}

	// 3rxx	skeq vr,xx	skip if register r = constant
	skeq1Instruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			y := uint8(opcode & 0x00FF)

			return fmt.Sprintf("skeq v%x, %d", vX, y)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			x := vm.registers[vX]
			y := uint8(opcode & 0x00FF)

			if x == y {
				vm.pc += 2 * InstructionSize
			} else {
				vm.pc += InstructionSize
			}

			return nil
		},
	}

	// 4rxx	skne vr,xx	skip if register r <> constant
	skne1Instruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			y := uint8(opcode & 0x00FF)

			return fmt.Sprintf("skne v%x, %d", vX, y)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			x := vm.registers[vX]
			y := uint8(opcode & 0x00FF)

			if x != y {
				vm.pc += 2 * InstructionSize
			} else {
				vm.pc += InstructionSize
			}

			return nil
		},
	}

	// 5ry0	skeq vr,vy	skip f register r = register y
	skeq2Instruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4

			return fmt.Sprintf("skeq v%x, v%x", vX, vY)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4
			x := vm.registers[vX]
			y := vm.registers[vY]

			if x == y {
				vm.pc += 2 * InstructionSize
			} else {
				vm.pc += InstructionSize
			}

			return nil
		},
	}

	// mov vr,xx	move constant to register r
	mov1Instruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			y := uint8(opcode & 0x00FF)

			return fmt.Sprintf("mov v%x, %d", vX, y)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			y := uint8(opcode & 0x00FF)

			vm.registers[vX] = y

			vm.pc += InstructionSize
			return nil
		},
	}

	// 7rxx	add vr,vx	add constant to register r	No carry generated
	add1Instruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			y := uint8(opcode & 0x00FF)

			return fmt.Sprintf("add v%x, %d", vX, y)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			y := uint8(opcode & 0x00FF)

			vm.registers[vX] += y

			vm.pc += InstructionSize
			return nil
		},
	}

	// 8ry0	mov vr,vy	move register vy into vr
	mov2Instruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4

			return fmt.Sprintf("mov v%x, v%x", vX, vY)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4
			y := vm.registers[vY]

			vm.registers[vX] = y

			vm.pc += InstructionSize
			return nil
		},
	}

	// 8ry1	or rx,ry	or register vy into register vx
	orInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4

			return fmt.Sprintf("or v%x, v%x", vX, vY)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4
			x := vm.registers[vX]
			y := vm.registers[vY]

			vm.registers[vX] = x | y

			vm.pc += InstructionSize
			return nil
		},
	}

	// 8ry2	and rx,ry	and register vy into register vx
	andInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4

			return fmt.Sprintf("and v%x, v%x", vX, vY)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4
			x := vm.registers[vX]
			y := vm.registers[vY]

			vm.registers[vX] = x & y

			vm.pc += InstructionSize
			return nil
		},
	}

	// 8ry3	xor rx,ry	exclusive or register ry into register rx
	xorInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4

			return fmt.Sprintf("xor v%x, v%x", vX, vY)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4
			x := vm.registers[vX]
			y := vm.registers[vY]

			vm.registers[vX] = x ^ y

			vm.pc += InstructionSize
			return nil
		},
	}

	// 8ry4	add vr,vy	add register vy to vr,carry in vf
	add2Instruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4

			return fmt.Sprintf("add v%x, v%x", vX, vY)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4
			x := vm.registers[vX]
			y := vm.registers[vY]

			vm.registers[vX] = x + y

			if vm.registers[vX] > 0xFF-vm.registers[vX] {
				vm.registers[0x0F] = 1
			} else {
				vm.registers[0x0F] = 0
			}

			vm.pc += InstructionSize
			return nil
		},
	}

	// 8ry5	sub vr,vy	subtract register vy from vr,borrow in vf	vf set to 1 if borrows
	subInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4

			return fmt.Sprintf("sub v%x, v%x", vX, vY)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4
			x := vm.registers[vX]
			y := vm.registers[vY]

			if y > x {
				vm.registers[0x0F] = 0
			} else {
				vm.registers[0x0F] = 1
			}

			vm.registers[vX] = x - y

			vm.pc += InstructionSize
			return nil
		},
	}

	// 8r06	shr vr	shift register vy right, bit 0 goes into register vf
	shrInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8

			return fmt.Sprintf("shr v%x", vX)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			x := vm.registers[vX]

			vm.registers[0x0F] = x & 0x1
			vm.registers[vX] = x >> 1
			vm.pc += InstructionSize
			return nil
		},
	}

	// 8ry7	rsb vr,vy	subtract register vr from register vy, result in vr	vf set to 1 if borrows
	rsbInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4

			return fmt.Sprintf("rsb v%x, v%x", vX, vY)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4
			x := vm.registers[vX]
			y := vm.registers[vY]

			if x > y {
				vm.registers[0x0F] = 0
			} else {
				vm.registers[0x0F] = 1
			}

			vm.registers[vX] = y - x
			vm.pc += InstructionSize

			return nil
		},
	}

	// 8r0e	shl vr	shift register vr left,bit 7 goes into register vf
	shlInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8

			return fmt.Sprintf("shl v%x", vX)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			x := vm.registers[vX]

			vm.registers[0x0F] = x >> 7
			vm.registers[vX] = x << 1

			vm.pc += InstructionSize

			return nil
		},
	}

	// 8r0e	shl vr	shift register vr left,bit 7 goes into register vf
	skne2Instruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4

			return fmt.Sprintf("skne v%x, v%x", vX, vY)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4
			x := vm.registers[vX]
			y := vm.registers[vY]

			if x != y {
				vm.pc += 2 * InstructionSize
			} else {
				vm.pc += InstructionSize
			}

			return nil
		},
	}

	// axxx	mvi xxx	Load index register with constant xxx
	mviInstruction = instruction{
		Name: func(opcode uint16) string {
			return fmt.Sprintf("mvi 0x%04x", opcode&0x0FFF)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vm.index = opcode & 0x0FFF
			vm.pc += InstructionSize

			return nil
		},
	}

	// bxxx	jmi xxx	Jump to address xxx+register v0
	jmiInstruction = instruction{
		Name: func(opcode uint16) string {
			return fmt.Sprintf("jmi 0x%04x", opcode&0x0FFF)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vm.pc = (opcode & 0x0FFF) + uint16(vm.registers[0])
			return nil
		},
	}

	// crxx	rand vr,xxx   	vr = random number less than or equal to xxx
	randInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			return fmt.Sprintf("rand v%x", vX)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			mask := uint16(opcode & 0x00FF)
			x := uint16(rand.IntN(256))
			x = x % (0xFF + 1)
			x = x & mask

			vm.registers[vX] = uint8(x)
			vm.pc += InstructionSize

			return nil
		},
	}

	// sprite rx,ry,s	Draw sprite at screen location rx,ry height s
	// Sprites stored in memory at location in index register, maximum 8 bits wide.
	// Wraps around the screen.
	// If when drawn, clears a pixel, vf is set to 1 otherwise it is zero.
	// All drawing is xor drawing (e.g. it toggles the screen pixels)
	spriteInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4
			height := opcode & 0x000F
			return fmt.Sprintf("sprite v%x, v%x, %d", vX, vY, height)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			vY := (opcode & 0x00F0) >> 4
			height := opcode & 0x000F

			xLocation, yLocation := uint16(vm.registers[vX]), uint16(vm.registers[vY])

			// slog.Debug(fmt.Sprintf("sprite v%x, v%x, %d", vX, vY, height))
			// slog.Debug(fmt.Sprintf("  sprite %d, %d, %d", xLocation, yLocation, height))

			hasCollision := uint8(0)
			for y := uint16(0); y < height; y++ {
				pixelAddr := y + vm.index
				if int(pixelAddr) >= len(vm.memory) {
					slog.Error("memory out of range",
						"addr", pixelAddr,
						"y", y,
						"index", vm.index,
						"index", fmt.Sprintf("0x%04x", vm.index),
					)
				}

				pixel := vm.memory[pixelAddr]

				const width = uint16(8)
				for x := uint16(0); x < width; x++ {
					mask := uint8(0x80 >> x)
					if (pixel & mask) != 0 {
						const stride = ScreenWidth
						screenAddr := getScreenAddr(x+xLocation, y+yLocation)

						if int(screenAddr) >= len(vm.gfx) {
							slog.Error("screen out of range",
								"addr", screenAddr,
								"stride", stride,
								"yLocation", yLocation,
								"y", y,
								"x", x,
								"xLocation", xLocation,
							)
						}

						if vm.gfx[screenAddr] != 0 {
							hasCollision = 1
						}

						vm.gfx[screenAddr] ^= 1
					}
				}
			}

			vm.registers[0x0F] = hasCollision
			vm.drawFlag = true
			vm.pc += InstructionSize

			return nil
		},
	}

	// ek9e	skpr k	skip if key (register rk) pressed	The key is a key number, see the chip-8 documentation
	skprInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			return fmt.Sprintf("skpr v%x", vX)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			x := vm.registers[vX]

			if vm.keypad[x] != 0 {
				vm.pc += 2 * InstructionSize
			} else {
				vm.pc += InstructionSize
			}

			return nil
		},
	}

	// eka1	skup k	skip if key (register rk) not pressed
	skupInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			return fmt.Sprintf("skup v%x", vX)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			x := vm.registers[vX]

			if vm.keypad[x] == 0 {
				vm.pc += 2 * InstructionSize
			} else {
				vm.pc += InstructionSize
			}

			return nil
		},
	}

	// fr07	gdelay vr	get delay timer into vr
	gdelayInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			return fmt.Sprintf("gdelay v%x", vX)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8

			vm.registers[vX] = vm.delayTimer
			vm.pc += InstructionSize
			return nil
		},
	}

	// fr0a	key vr	wait for for keypress,put key in register vr
	keyInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			return fmt.Sprintf("key v%x", vX)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			keyPressed := false

			for i := range vm.keypad {
				if vm.keypad[i] != 0 {
					vm.registers[vX] = uint8(i)
					keyPressed = true
				}
			}

			if !keyPressed {
				return nil
			}

			vm.pc += InstructionSize
			return nil
		},
	}

	// fr15	sdelay vr	set the delay timer to vr
	sdelayInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			return fmt.Sprintf("sdelay v%x", vX)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8

			vm.delayTimer = vm.registers[vX]
			vm.pc += InstructionSize
			return nil
		},
	}

	// fr18	ssound vr	set the sound timer to vr
	ssoundInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			return fmt.Sprintf("ssound v%x", vX)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8

			vm.soundTimer = vm.registers[vX]
			vm.pc += InstructionSize
			return nil
		},
	}

	// fr1e	adi vr	add register vr to the index register
	adiInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			return fmt.Sprintf("ssound v%x", vX)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			x := uint16(vm.registers[vX])

			if vm.index+x > 0x0FFF {
				vm.registers[0x0F] = 1
			} else {
				vm.registers[0x0F] = 0
			}

			vm.index += x
			vm.pc += InstructionSize
			return nil
		},
	}

	// fr29	font vr	point I to the sprite for hexadecimal character in vr	Sprite is 5 bytes high
	fontInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			return fmt.Sprintf("font v%x", vX)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			x := uint16(vm.registers[vX])
			x = x * 0x5
			vm.index = x
			vm.pc += InstructionSize
			return nil
		},
	}

	// fr33	bcd vr	store the bcd representation of register vr at location I,I+1,I+2	Doesn't change I
	bcdInstruction = instruction{
		Name: func(opcode uint16) string {
			vX := (opcode & 0x0F00) >> 8
			return fmt.Sprintf("bcd v%x", vX)
		},
		Execute: func(vm *VM, opcode uint16) error {
			vX := (opcode & 0x0F00) >> 8
			x := vm.registers[vX]

			vm.memory[vm.index] = x / 100
			vm.memory[vm.index+1] = (x / 10) % 10
			vm.memory[vm.index+2] = x % 10
			vm.pc += InstructionSize
			return nil
		},
	}

	// fr55	str v0-vr	store registers v0-vr at location I onwards	I is incremented to point to the next location on. e.g. I = I + r + 1
	strInstruction = instruction{
		Name: func(opcode uint16) string {
			n := (opcode & 0x0F00) >> 8
			return fmt.Sprintf("str %d", n)
		},
		Execute: func(vm *VM, opcode uint16) error {
			n := (opcode & 0x0F00) >> 8

			for i := uint16(0); i <= n; i++ {
				vm.memory[vm.index+i] = vm.registers[i]
			}

			// On the original interpreter, when the operation is done, I = I + X + 1.
			vm.index += n + 1

			vm.pc += InstructionSize
			return nil
		},
	}

	// fx65	ldr v0-vr	load registers v0-vr from location I onwards.
	ldrInstruction = instruction{
		Name: func(opcode uint16) string {
			n := (opcode & 0x0F00) >> 8
			return fmt.Sprintf("ldr %d", n)
		},
		Execute: func(vm *VM, opcode uint16) error {
			n := (opcode & 0x0F00) >> 8

			for i := uint16(0); i <= n; i++ {
				vm.registers[i] = vm.memory[vm.index+i]
			}

			// On the original interpreter, when the operation is done, I = I + X + 1.
			vm.index += n + 1

			vm.pc += InstructionSize
			return nil
		},
	}

	unknownInstruction = instruction{
		Name: func(opcode uint16) string {
			return fmt.Sprintf("unknown 0x%04X", opcode)
		},
		Execute: func(vm *VM, opcode uint16) error {
			return fmt.Errorf("unknown op code 0x%04X", opcode)
		},
	}
)

func getScreenAddr(x, y uint16) uint16 {
	x %= ScreenWidth
	y %= ScreenHeight

	screenAddr := ScreenWidth*(y) + x
	return screenAddr
}
