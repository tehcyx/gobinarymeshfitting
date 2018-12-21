package main

func main() {
	var renderInput *RenderInput

	renderInput = new(RenderInput)

	if CoreInit(renderInput) {
		return
	}
	CoreRun(renderInput)
	CoreCleanup()
}
