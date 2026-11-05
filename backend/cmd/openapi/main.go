package main

import (
	"flag"
	"log"
	"os"

	"courserep/backend/internal/httpapi"
)

func main() {
	data, err := httpapi.OpenAPIJSON()
	if err != nil {
		log.Fatal(err)
	}
	out := flag.String("out", "", "write the generated OpenAPI JSON to this path; stdout when empty")
	flag.Parse()
	if *out != "" {
		if err = os.WriteFile(*out, data, 0644); err != nil {
			log.Fatal(err)
		}
		return
	}
	if _, err = os.Stdout.Write(data); err != nil {
		log.Fatal(err)
	}
}
