package main

import "github.com/hahaxo/leto"

func main() {
	if err := leto.GeneratePage("view/page/page.html", "output.html", nil); err != nil {
		panic(err)
	}
}
