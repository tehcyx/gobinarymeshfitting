package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-gl/gl/v2.1/gl"
)

const (
	// OPTIONS

	fillModeFill = 0
	fillModeBoth = 1
	fillModeWire = 2

	quads           = false
	smoothNormals   = false
	defaultFillMode = 1
	flatQuads       = false

	specularPower = 0.0

	fastGrouping = false
)

func shaderErrorCheck(shader uint32, text string) {
	var success int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &success)
	if success == gl.FALSE {
		fmt.Printf("Failed to compile %s\n", text)
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		fmt.Println(fmt.Errorf("failed to compile: %v", log))
	}
}

func linkerErrorCheck(shader uint32, text string) {
	var success int32
	gl.GetProgramiv(shader, gl.LINK_STATUS, &success)
	if success == gl.FALSE {
		fmt.Printf("Failed to link %s\n", text)
		var logLength int32
		gl.GetProgramiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(shader, logLength, nil, gl.Str(log))

		fmt.Println(fmt.Errorf("failed to link program: %v", log))
	}
}

type DebugScene struct {
	lastSpace      int
	outlineVisible bool
	fillmode       int
	cull           bool
	quads          bool
	smoothShading  bool
	flatQuads      bool
	guiVisible     bool
	updateFocus    bool
	lineWidth      float32
	lineColor      [4]float32
	fillColor      [4]float32
	clearColor     [4]float32
	specularPower  float32
	pointsVbo      uint32
	colorsVbo      uint32
	vao            uint32
	ibo            uint32

	vertexShader   uint32
	fragmentShader uint32
	shaderProgram  uint32

	outlineVs uint32
	outlineFs uint32
	outlineSp uint32

	shaderProjection    int32
	shaderView          int32
	shaderMulClr        int32
	shaderEyePos        int32
	shaderSmoothShading int32
	shaderSpecularPower int32

	outlineShaderProjection int32
	outlineShaderView       int32
	outlineShaderMulClr     int32

	camera    FPSCamera
	dualChunk *Chunk
	glChunk   GLChunk
	world     WorldOctree

	binaryChunk *BinaryChunk

	glMutex        *sync.Mutex
	lastExtraction int // clock_t ???
	updateRequired bool
	updateMutex    *sync.Mutex
}

func NewDebugScene(renderInput *RenderInput) *DebugScene {
	ds := new(DebugScene)
	ds.lastSpace = 0
	ds.outlineVisible = false
	ds.smoothShading = smoothNormals
	ds.fillmode = defaultFillMode
	ds.quads = quads
	ds.flatQuads = flatQuads
	ds.cull = true
	ds.guiVisible = true
	ds.updateFocus = true
	ds.lineWidth = 1.0
	ds.specularPower = specularPower
	ds.fillColor[0] = 0.85
	ds.fillColor[1] = 0.85
	ds.fillColor[2] = 0.85
	ds.fillColor[3] = 1.0

	ds.lineColor[0] = 0.25
	ds.lineColor[1] = 0.25
	ds.lineColor[2] = 0.25
	ds.lineColor[3] = 1.0

	ds.clearColor[0] = 0.0
	ds.clearColor[1] = 0.5
	ds.clearColor[2] = 1.0
	ds.clearColor[3] = 1.0

	dx := []float32{0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0}
	dy := []float32{0.0, 0.0, 1.0, 1.0, 0.0, 0.0, 1.0, 1.0}
	dz := []float32{0.0, 0.0, 0.0, 0.0, 1.0, 1.0, 1.0, 1.0}

	var vertexShader = `
	#version 400 core
	attribute vec3 vertex_position;
	attribute vec3 vertex_normal;
	attribute vec3 vertex_color;
	uniform mat4 projection;
	uniform mat4 view;
	uniform vec3 mul_color;
	uniform float smooth_shading;
	uniform float specular_power;
	out vec3 f_normal;
	out vec3 f_color;
	out vec3 f_mul_color;
	out vec3 f_ec_pos;
	out float f_smooth_shading;
	out float f_specular_power;
	out float log_z;
	void main() {
		f_normal = normalize(vertex_normal);
		f_color = vertex_color;
		f_mul_color = mul_color;
		f_smooth_shading = smooth_shading;
		f_specular_power = specular_power;
		f_ec_pos = vertex_position;
		const float near = 0.00001;
		const float far = 10000.0;
		const float C = 0.001;
		gl_Position = projection * view * vec4(vertex_position, 1);
		const float FC = 1.0f / log(far * C + 1.0);
		log_z = log(gl_Position.w * C + 1.0) * FC;
		gl_Position.z = (2.0 * log_z - 1.0) * gl_Position.w;
	}
	` + "\x00"

	var fragmentShader = `
	#version 400 core
	in vec3 f_normal;
	in vec3 f_color;
	in vec3 f_mul_color;
	in vec3 f_ec_pos;
	in float f_smooth_shading;
	in float f_specular_power;
	in float log_z;
	out vec4 frag_color;
	void main() {
		vec3 normal;
		if (f_smooth_shading != 0.0)
			normal = f_normal;
		else
			normal = normalize(cross(dFdx(f_ec_pos), dFdy(f_ec_pos)));
		float d = dot(normalize(-vec3(0.1, -1.0, 0.5)), normal);
		float m = mix(0.2, 1.0, d * 0.5 + 0.5);
		float s = (f_specular_power > 0.0 ? pow(max(0.0, d), f_specular_power) : 0.0);
		vec3 color = vec3(0.3, 0.3, 0.5);
		vec3 color2 = vec3(0.1, 0.1, 0.25);
		vec3 result = f_color * m + f_color * s;
		frag_color = vec4(result, 1.0);
		gl_FragDepth = log_z;
	}
	` + "\x00"

	var outlineVs = `
	#version 400 core
	attribute vec3 vertex_position;
	uniform mat4 projection;
	uniform mat4 view;
	uniform vec3 mul_color;
	out vec3 f_mul_color;
	out float log_z;
	void main() {
		f_mul_color = mul_color;
		gl_Position = projection * view * vec4(vertex_position, 1);
		const float near = 0.00001;
		const float far = 10000.0;
		const float C = 0.001;
		const float FC = 1.0 / log(far * C + 1.0);
		log_z = log(gl_Position.w * C + 1.0) * FC;
		gl_Position.z = (2.0 * log_z - 1.0) * gl_Position.w;
	}
	` + "\x00"

	var outlineFs = `
	#version 400 core
	in vec3 f_mul_color;
	in float log_z;
	out vec4 frag_color;
	void main() {
		frag_color = vec4(f_mul_color, 1.0);
		gl_FragDepth = log_z - 0.00001;
	}
	` + "\x00"

	var csources **uint8
	var free func()

	ds.vertexShader = gl.CreateShader(gl.VERTEX_SHADER)
	csources, free = gl.Strs(vertexShader)
	gl.ShaderSource(ds.vertexShader, 1, csources, nil)
	free()
	gl.CompileShader(ds.vertexShader)
	shaderErrorCheck(ds.vertexShader, "regular vertex shader")

	ds.fragmentShader = gl.CreateShader(gl.FRAGMENT_SHADER)
	csources, free = gl.Strs(fragmentShader)
	gl.ShaderSource(ds.fragmentShader, 1, csources, nil)
	free()
	gl.CompileShader(ds.fragmentShader)
	shaderErrorCheck(ds.fragmentShader, "regular fragment shader")

	ds.shaderProgram = gl.CreateProgram()
	gl.AttachShader(ds.shaderProgram, ds.vertexShader)
	gl.AttachShader(ds.shaderProgram, ds.fragmentShader)

	gl.BindAttribLocation(ds.shaderProgram, 0, gl.Str("vertex_position"))
	gl.BindAttribLocation(ds.shaderProgram, 1, gl.Str("vertex_normal"))

	gl.LinkProgram(ds.shaderProgram)
	linkerErrorCheck(ds.shaderProgram, "regular shader")

	ds.shaderProjection = gl.GetUniformLocation(ds.shaderProgram, gl.Str("projection"))
	ds.shaderView = gl.GetUniformLocation(ds.shaderProgram, gl.Str("view"))
	ds.shaderMulClr = gl.GetUniformLocation(ds.shaderProgram, gl.Str("mul_color"))
	ds.shaderEyePos = gl.GetUniformLocation(ds.shaderProgram, gl.Str("eye_pos"))
	ds.shaderSmoothShading = gl.GetUniformLocation(ds.shaderProgram, gl.Str("smooth_shading"))
	ds.shaderSpecularPower = gl.GetUniformLocation(ds.shaderProgram, gl.Str("specular_power"))

	// outline:

	ds.outlineVs = gl.CreateShader(gl.VERTEX_SHADER)
	csources, free = gl.Strs(outlineVs)
	gl.ShaderSource(ds.outlineVs, 1, csources, nil)
	free()
	gl.CompileShader(ds.outlineVs)
	shaderErrorCheck(ds.outlineVs, "outline vertex shader")

	ds.outlineFs = gl.CreateShader(gl.FRAGMENT_SHADER)
	csources, free = gl.Strs(outlineFs)
	gl.ShaderSource(ds.outlineFs, 1, csources, nil)
	free()
	gl.CompileShader(ds.outlineFs)
	shaderErrorCheck(ds.outlineFs, "outline fragment shader")

	ds.outlineSp = gl.CreateProgram()
	gl.AttachShader(ds.outlineSp, ds.outlineVs)
	gl.AttachShader(ds.outlineSp, ds.outlineFs)

	gl.BindAttribLocation(ds.outlineSp, 0, gl.Str("vertex_position"))

	gl.LinkProgram(ds.outlineSp)
	linkerErrorCheck(ds.outlineSp, "outline shader")

	ds.outlineShaderProjection = gl.GetUniformLocation(ds.outlineSp, gl.Str("projection"))
	ds.outlineShaderView = gl.GetUniformLocation(ds.outlineSp, gl.Str("view"))
	ds.outlineShaderMulClr = gl.GetUniformLocation(ds.outlineSp, gl.Str("mul_color"))

	ds.camera.init(renderInput.winWidth, renderInput.winHeight, renderInput)
	ds.camera.setShader(ds.shaderProjection, ds.shaderView)

	fmt.Printf("here would be UI initialization...\n")
	// printf("Initializing imgui...");
	// ImGui_ImplGlfwGL3_Init(render_input->window, true);
	// ImGui::StyleColorsLight();
	// ImGui::GetStyle().Colors[ImGuiCol_WindowBg].w = 0.85f;
	// printf("Done.\n");

	ds.initWorld()

	return ds
}

func (ds *DebugScene) initSingleChunk() {

}

func (ds *DebugScene) initBinaryChunk() {

}

func (ds *DebugScene) initWorld() {

}

func (ds *DebugScene) update(input *RenderInput) int {
	return 0
}

func (ds *DebugScene) render(input *RenderInput) int {
	return 0
}

func (ds *DebugScene) renderSingleChunk() {

}

func (ds *DebugScene) renderBinaryChunk() {

}

func (ds *DebugScene) renderWorld() {

}

func (ds *DebugScene) keyCallback(key, scancode, action, mods int) {

}

func (ds *DebugScene) renderGui() {

}

func (ds *DebugScene) destroy() {
	ds.world.watcher.stop()
	ds.world.watcher.generator.stop()
}
