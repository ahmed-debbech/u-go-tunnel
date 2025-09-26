package main

import (
	"fmt"
	"log"
	"net/http"
)

var indexHTML = `
<!DOCTYPE html>
<html>
<head>
  <title>Go Static Server</title>
  <style>
    body { font-family: Arial, sans-serif; background: #f2f2f2; text-align: center; }
    h1 { color: #333; }
  </style>
</head>
<body>
  <h1>Hello from Go 🚀</h1>
  <p>This page is served from a Go string variable!</p>
  <script>
    console.log("JS also works ✅");
  </script>
</body>
</html>
`

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, indexHTML)
	})

	log.Println("Server running at http://localhost:8080/")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
