package main

type WorldOctree struct {
	watcher Watcher
}

type Watcher struct {
	generator Generator
}

type Generator struct {
}

func (w *Watcher) stop() {

}

func (g *Generator) stop() {

}
