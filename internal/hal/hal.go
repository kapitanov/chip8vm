package hal

import (
	"errors"
	"fmt"
	"log/slog"
	"time"
	"unsafe"

	"github.com/kapitanov/chip8vm/internal/vm"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	WindowWidth  = 1024
	WindowHeight = 512
)

type HAL struct {
	window          *sdl.Window
	renderer        *sdl.Renderer
	texture         *sdl.Texture
	backBuffer      []uint32
	backBufferPitch int
}

var (
	ErrReboot = errors.New("reboot")
	ErrQuit   = errors.New("quit")
)

func New() (*HAL, error) {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		return nil, fmt.Errorf("failed to init sdl: %w", err)
	}

	window, err := sdl.CreateWindow("CHIP-8", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, WindowWidth, WindowHeight, sdl.WINDOW_SHOWN|sdl.WINDOW_UTILITY)
	if err != nil {
		return nil, fmt.Errorf("failed to create sdl window: %w", err)
	}
	slog.Debug("hal: create window")
	window.Show()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		return nil, fmt.Errorf("failed to create sdl renderer: %w", err)
	}
	err = renderer.SetLogicalSize(WindowWidth, WindowHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to resize sdl renderer: %w", err)
	}
	slog.Debug("hal: create renderer")

	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_ARGB8888, sdl.TEXTUREACCESS_STREAMING, vm.ScreenWidth, vm.ScreenHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to create sdl texture: %w", err)
	}
	slog.Debug("hal: create texture")

	return &HAL{
		window:          window,
		renderer:        renderer,
		texture:         texture,
		backBuffer:      make([]uint32, vm.ScreenWidth*vm.ScreenHeight),
		backBufferPitch: int(vm.ScreenWidth) * int(unsafe.Sizeof(uint32(0))),
	}, nil
}

func (hal *HAL) Shutdown() {
	if err := hal.texture.Destroy(); err != nil {
		slog.Error("failed to destroy sdl texture", "err", err)
	}

	if err := hal.renderer.Destroy(); err != nil {
		slog.Error("failed to destroy sdl renderer", "err", err)
	}

	if err := hal.window.Destroy(); err != nil {
		slog.Error("failed to destroy sdl window", "err", err)
	}

	sdl.Quit()
}

func (hal *HAL) ReadInput(keyDown func(vm.Key), keyUp func(vm.Key)) error {
	for e := sdl.PollEvent(); e != nil; e = sdl.PollEvent() {
		switch e.GetType() {
		case sdl.QUIT:
			slog.Debug("hal: exit requested")
			return ErrQuit
		case sdl.KEYDOWN:
			err := hal.processKeyDown(e.(*sdl.KeyboardEvent), keyDown)
			if err != nil {
				return err
			}

		case sdl.KEYUP:
			hal.processKeyUp(e.(*sdl.KeyboardEvent), keyUp)
		}
	}

	return nil
}

func (hal *HAL) processKeyDown(e *sdl.KeyboardEvent, callback func(vm.Key)) error {
	if e.Keysym.Scancode == sdl.SCANCODE_BACKSPACE {
		return ErrReboot
	}

	key, ok := keyMap(e)
	if ok {
		callback(key)
	}

	return nil
}

func (hal *HAL) processKeyUp(e *sdl.KeyboardEvent, callback func(vm.Key)) {
	key, ok := keyMap(e)
	if ok {
		callback(key)
	}
}

func keyMap(e *sdl.KeyboardEvent) (vm.Key, bool) {
	// Physical                Logical
	// ================        =================
	// | 1 | 2 | 3 | 4 |       | 1 | 2 | 3 | C |
	// | q | w | e | r |       | 4 | 5 | 6 | D |
	// | a | s | d | e |  <=>  | 7 | 8 | 9 | E |
	// | z | x | c | v |       | A | 0 | B | F |
	// ================        =================

	switch e.Keysym.Scancode {
	case sdl.SCANCODE_X:
		return vm.Key0, true
	case sdl.SCANCODE_1:
		return vm.Key1, true
	case sdl.SCANCODE_2:
		return vm.Key2, true
	case sdl.SCANCODE_3:
		return vm.Key3, true
	case sdl.SCANCODE_Q:
		return vm.Key4, true
	case sdl.SCANCODE_W:
		return vm.Key5, true
	case sdl.SCANCODE_E:
		return vm.Key6, true
	case sdl.SCANCODE_A:
		return vm.Key7, true
	case sdl.SCANCODE_S:
		return vm.Key8, true
	case sdl.SCANCODE_D:
		return vm.Key9, true
	case sdl.SCANCODE_Z:
		return vm.KeyA, true
	case sdl.SCANCODE_C:
		return vm.KeyB, true
	case sdl.SCANCODE_4:
		return vm.KeyC, true
	case sdl.SCANCODE_R:
		return vm.KeyD, true
	case sdl.SCANCODE_F:
		return vm.KeyE, true
	case sdl.SCANCODE_V:
		return vm.KeyF, true
	default:
		return 0, false
	}
}

func (hal *HAL) Draw(gfx []uint8) error {
	const (
		bgColor = uint32(0x000000)
		fgColor = uint32(0xbea700)
	)

	for y := 0; y < vm.ScreenHeight; y++ {

		for x := 0; x < vm.ScreenWidth; x++ {
			i := x + y*vm.ScreenWidth

			color := bgColor
			if gfx[i] != 0 {
				color = fgColor
			}

			hal.backBuffer[i] = color
		}
	}

	backBufferPtr := unsafe.Pointer(&hal.backBuffer[0])
	if err := hal.texture.Update(nil, backBufferPtr, hal.backBufferPitch); err != nil {
		return fmt.Errorf("failed to update sdl texture: %w", err)
	}

	if err := hal.renderer.Clear(); err != nil {
		return fmt.Errorf("failed to clear sdl renderer: %w", err)
	}

	if err := hal.renderer.Copy(hal.texture, nil, nil); err != nil {
		return fmt.Errorf("failed to copy sdl texture to renderer: %w", err)
	}

	hal.renderer.Present()
	hal.window.SetAlwaysOnTop(true)
	return nil
}

func (hal *HAL) WaitForNextFrame() error {
	const delayDuration = 1200 * time.Microsecond
	time.Sleep(delayDuration)
	return nil
}

func (hal *HAL) WaitForQuit() error {
	for {
		for e := sdl.PollEvent(); e != nil; e = sdl.PollEvent() {
			if e.GetType() == sdl.QUIT {
				return nil
			}
		}
	}
}
