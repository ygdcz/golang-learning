package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
)

const (
	UPLOAD_DIR = "./uploads"
	ListDir    = 0x0001
)

var templates = make(map[string]*template.Template, 3)

func init() {
	fileInfoArr, err := os.ReadDir("views")
	if err != nil {
		panic(err)
	}
	for _, fileInfo := range fileInfoArr {
		// 去掉 .html 扩展名作为模板键名
		tmplName := fileInfo.Name()
		if len(tmplName) > 5 && tmplName[len(tmplName)-5:] == ".html" {
			tmplName = tmplName[:len(tmplName)-5]
		}
		templates[tmplName] = template.Must(template.ParseFiles("views/" + fileInfo.Name()))
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		check(renderHtml(w, "upload", nil))
	}
	if r.Method == "POST" {
		f, h, err := r.FormFile("image")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		filename := h.Filename
		defer f.Close()
		t, err := os.Create(UPLOAD_DIR + "/" + filename)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer t.Close()
		if _, err := io.Copy(t, f); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/view?id="+filename, http.StatusFound)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	// imageId := r.URL.Query().Get("id")
	imageId := r.FormValue("id")
	imagePath := UPLOAD_DIR + "/" + imageId
	if _, err := os.Stat(imagePath); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image")
	http.ServeFile(w, r, imagePath)
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	fileInfoArr, err := os.ReadDir(UPLOAD_DIR)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	locals := make(map[string]any)
	images := []string{}
	for _, fileInfo := range fileInfoArr {
		images = append(images, fileInfo.Name())
	}
	locals["images"] = images
	check(renderHtml(w, "list", locals))
}

func staticDirHandler(mux *http.ServeMux, prefix string, staticDir string, flags int) {
	mux.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
		file := staticDir + r.URL.Path[len(prefix):]
		if (flags & ListDir) == 0 {
			http.ServeFile(w, r, file)
		}
	})
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func renderHtml(w http.ResponseWriter, tmpl string, locals map[string]any) error {
	if templates[tmpl] == nil {
		return fmt.Errorf("template %s not found", tmpl)
	}
	return templates[tmpl].Execute(w, locals)
}

func safeHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Println(err)
				http.Error(w, http.StatusText(500), 500)
			}
		}()
		fn(w, r)
	}
}

func main() {
	mux := http.NewServeMux()
	staticDirHandler(mux, "/assets", "assets", 0)
	mux.HandleFunc("/upload", safeHandler(uploadHandler))
	mux.HandleFunc("/view", safeHandler(viewHandler))
	mux.HandleFunc("/", safeHandler(listHandler))
	log.Println("Listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err.Error())
	}
}
