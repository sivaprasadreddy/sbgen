package main

import (
	"log"

	"github.com/egoist/mygo"
)

func main() {
	// AppGeneratorService's methods are callable from the frontend:
	// `mygo generate` turns them into typed TypeScript functions in src/mygo.ts.
	mygo.Bind(&AppGeneratorService{})
	// Templates and defaults live in the user data directory, e.g.
	// ~/Library/Application Support/SBGen/templates.json.
	userData := func() (string, error) { return mygo.App.Path(mygo.PathUserData) }
	mygo.Bind(NewTemplateService(userData))
	mygo.Bind(NewDefaultsService(userData))
	mygo.Bind(NewDialogService())

	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:     "SBGen",
			Width:     1200,
			Height:    800,
			MinWidth:  640,
			MinHeight: 560,
			// The page's background in src/style.css.
			BackgroundColor: "#f6f4fa",
			// The page extends under the window buttons; the sidebar drags the window.
			TitleBarStyle: mygo.TitleBarHiddenInset,
			// Opens where the user left it last time.
			StateKey: "main",
			// The frontend: devUrl during `mygo dev`, frontendDist
			// embedded by `mygo build` otherwise (see mygo.config.ts).
			URL: "/",
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
