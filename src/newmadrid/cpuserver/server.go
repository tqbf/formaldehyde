package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func compile(path string, haml string) (err error) {
	err = filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if strings.HasSuffix(file, ".haml") {
			raw := strings.Replace(file, ".haml", ".html", -1)
			var rawinfo os.FileInfo
			rebuild := true

			if rawinfo, err = os.Stat(raw); err == nil {
				if rawinfo.ModTime().Before(info.ModTime()) {
					rebuild = false
				}
			}

			if rebuild {
				cmd := exec.Command(haml, file)
				bytes, err := cmd.Output()
				if err != nil {
					return err
				}

				f, err := os.Create(raw)
				if err != nil {
					return err
				}
				defer f.Close()

				if _, err = f.Write(bytes); err != nil {
					return err
				}
			}
		}

		return nil
	})

	return
}

func wireStatics(path string) {
	suffixes := []string{
		".js",
		".png",
		".jpg",
		".css",
	}

	filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		for _, suf := range suffixes {
			if strings.HasSuffix(file, suf) {
				http.HandleFunc(fmt.Sprintf("/%s", filepath.Base(file)),
					func(w http.ResponseWriter, r *http.Request) {
						switch suf {
						case ".png":
							fallthrough
						case ".jpg":
							w.Header().Set("Content-type", "image")
						case ".js":
							w.Header().Set("Content-type", "text/javascript")
						case ".css":
							w.Header().Set("Content-type", "text/css")
						}

						http.ServeFile(w, r, file)
					})
				break
			}
		}

		return nil
	})
}

func main() {
	vroot := flag.String("root", "data/newmadrid/views", "Path to HTML/JS templates")
	haml := flag.String("haml", "/usr/bin/haml", "Path to Haml command")

	flag.Parse()

	if err := compile(*vroot, *haml); err != nil {
		log.Fatal(err)
	}

	wireStatics(*vroot)

	http.Handle("/", CpuInterface(*vroot))

	log.Fatal(http.ListenAndServe(":8080", nil))
}