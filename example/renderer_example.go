package main

import "github.com/hahaxo/leto"

func main() {
	type Home struct {
		Title string
	}

	data_index := Home{Title: "home"}
	data_about := Home{Title: "about"}

	r, err := leto.New("view")
	if err != nil {
		panic(err)
	}

	r.Render("index", "dist/index.html", data_index)
	r.Render("about", "dist/about.html", data_about)

}
