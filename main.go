package main

import "github.com/zserge/webview"
import "github.com/getlantern/systray"

func main() {
	systray.Run(onReady,onExit)
	// Open wikipedia in a 800x600 resizable window
	webview.Open("Minimal webview example",
		"https://www.baidu.com", 800, 600, true)
}


func onReady() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("Awesome App")
	systray.SetTooltip("Pretty awesome超级棒")
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	// Sets the icon of a menu item. Only available on Mac.
	mQuit.SetIcon(icon.Data)
}

func onExit() {
	// clean up here
}