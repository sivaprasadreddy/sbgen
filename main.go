package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	// Templates and defaults live in the user data directory, e.g.
	// ~/Library/Application Support/SBGen/templates.json.
	userData := func() (string, error) { return mygo.App.Path(mygo.PathUserData) }

	mygo.App.WhenReady(func() {
		var win *mygo.Window
		// Generation runs Maven or Gradle, so it runs off the UI thread and
		// reports back on it.
		async := func(work func() func()) {
			go func() {
				apply := work()
				win.Update(apply)
			}()
		}
		app := newWizard(NewTemplateService(userData), NewDefaultsService(userData), (&AppGeneratorService{}).GenerateProject, async)
		win = mygo.NewWindow(mygo.WindowOptions{
			Title:     "SBGen",
			Width:     1200,
			Height:    800,
			MinWidth:  900,
			MinHeight: 640,
			// The sidebar runs under the window buttons and drags the window.
			TitleBarStyle: mygo.TitleBarHiddenInset,
			// Opens where the user left it last time.
			StateKey: "main",
			// The window shows the interface MyGo draws, not a web page.
			Content: ui.View(app.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
