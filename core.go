package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/go-gl/gl/v2.1/gl"
	"gopkg.in/veandco/go-sdl2.v0/sdl"
)

type RenderInput struct {
	window      *sdl.Window
	context     sdl.GLContext
	winWidth    int
	winHeight   int
	delta       float32
	shouldClose bool
}

var globalScene *DebugScene

const (
	winTitle         = "OpenGL Shader"
	defaultWinWidth  = 800
	defaultWinHeight = 600

	maxFrameSkip     = 1
	updatesPerSecond = 60
	skipTicks        = 1 / updatesPerSecond
)

func CoreInit(out *RenderInput) bool {
	var err error

	runtime.LockOSThread()
	if err = sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		return false
	}
	defer sdl.Quit()

	sdl.GL_SetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)
	sdl.GL_SetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 3)
	sdl.GL_SetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 3)

	// init window
	out.window, err = sdl.CreateWindow(winTitle, sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED,
		defaultWinWidth, defaultWinHeight, sdl.WINDOW_OPENGL)
	out.winHeight = defaultWinHeight
	out.winWidth = defaultWinWidth
	out.shouldClose = false
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create window: %s\n", err)
		return false
	}
	defer out.window.Destroy()

	// init context
	out.context, err = sdl.GL_CreateContext(out.window)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create context: %s\n", err)
		return false
	}
	defer sdl.GL_DeleteContext(out.context)

	// init gl
	if err := gl.Init(); err != nil {
		panic(err)
	}

	// print renderer and gl version
	renderer := gl.GoStr(gl.GetString(gl.RENDERER))
	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("Renderer", renderer)
	fmt.Println("OpenGL version", version)

	// enable depth testing
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)

	return true
}

func CoreRun(render *RenderInput) bool {
	scene := NewDebugScene(render)
	globalScene = scene

	lastTime := sdl.GetTicks()
	timer := lastTime
	currentTime := uint32(0)
	nextTick := lastTime

	frameCounter := 0
	lastFps := 0
	updateCounter := 0
	tupdateCounter := 0

	for !render.shouldClose {
		currentTime = sdl.GetTicks()
		render.delta = float32(currentTime - lastTime)
		lastTime = currentTime
		updateCounter = 0

		for sdl.GetTicks() > nextTick && updateCounter < maxFrameSkip {
			scene.update(render)
			updateCounter++
			nextTick += skipTicks
			tupdateCounter++
		}

		scene.render(render)

		frameCounter++

		if sdl.GetTicks()-timer > 1.0 {
			timer++
			lastFps = frameCounter
			fmt.Printf("FPS: %d, Updates: %d\n", lastFps, tupdateCounter)
			frameCounter = 0
			tupdateCounter = 0
		}
	}
	return false
}

func CoreCleanup() {
	sdl.Quit()
}

func keyCallback(window *sdl.Window, key, scancode, action, mods int) {
	globalScene.keyCallback(key, scancode, action, mods)
}
