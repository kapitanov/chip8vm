.PHONY: build run run-rom run-test-rom download-roms download-rom

build:
	@mkdir -p ./bin
	@[ -f ./bin/chip8vm ] && rm ./bin/chip8vm || true
	go build -o ./bin/chip8vm


run:
	@if [ -z "$(rom)" ]; then echo "Usage: make run ROM=<rom>"; exit 1; fi
	@make build
	./bin/chip8vm ./roms/$(rom)

run-test-rom:
	@make run-rom rom=test_opcode.ch8

download-roms:
	@make download-rom URL=https://github.com/corax89/chip8-test-rom/raw/refs/heads/master/test_opcode.ch8
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/15PUZZLE
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/BLINKY
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/BRIX
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/CONNECT4
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/GUESS
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/HIDDEN
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/INVADERS
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/KALEID
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/MAZE
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/MERLIN
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/MISSILE
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/PONG
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/PONG2
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/PUZZLE
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/SYZYGY
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/TANK
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/TETRIS
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/TICTAC
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/UFO
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/VBRIX
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/VERS
	@make download-rom URL=https://github.com/JamesGriffin/CHIP-8-Emulator/raw/refs/heads/master/roms/WIPEOFF

download-rom:
	@FILENAME=$(shell basename "$(URL)"); \
	curl -sL "$(URL)" -o ./roms/$$FILENAME; \
	echo "Downloaded $$FILENAME";
