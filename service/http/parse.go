package http

import (
	"strings"
	"net/http"
	"os"
	"log"
)

// MaxLevel defines maximum tree depth for incoming request data and files.
const MaxLevel = 127

type dataTree map[string]interface{}
type fileTree map[string]interface{}

// parseData parses incoming request body into data tree.
func parseData(r *http.Request) (dataTree, error) {
	data := make(dataTree)
	if r.PostForm != nil {
		for k, v := range r.PostForm {
			data.push(k, v)
		}
	}

	if r.MultipartForm != nil {
		for k, v := range r.MultipartForm.Value {
			data.push(k, v)
		}
	}

	return data, nil
}

// pushes value into data tree.
func (d dataTree) push(k string, v []string) {
	if len(v) == 0 {
		// skip empty values
		return
	}

	indexes := make([]string, 0)
	for _, index := range strings.Split(strings.Replace(k, "]", "", MaxLevel), "[") {
		indexes = append(indexes, index)
	}

	if len(indexes) <= MaxLevel {
		d.mount(indexes, v)
	}
}

// mount mounts data tree recursively.
func (d dataTree) mount(i []string, v []string) {
	log.Println(i, ">", v)

	if len(v) == 0 {
		return
	}

	if len(i) == 1 {
		// single value context
		d[i[0]] = v[0]
		return
	}

	if len(i) == 2 && i[1] == "" {
		// non associated array of elements
		d[i[0]] = v
		return
	}

	if p, ok := d[i[0]]; ok {
		p.(dataTree).mount(i[1:], v)
	}

	d[i[0]] = make(dataTree)
	d[i[0]].(dataTree).mount(i[1:], v)
}

// parse incoming dataTree request into JSON (including contentMultipart form dataTree)
func parseUploads(r *http.Request, cfg *UploadsConfig) (*Uploads, error) {
	u := &Uploads{
		cfg:  cfg,
		tree: make(fileTree),
		list: make([]*FileUpload, 0),
	}

	for k, v := range r.MultipartForm.File {
		files := make([]*FileUpload, 0, len(v))
		for _, f := range v {
			files = append(files, NewUpload(f))
		}

		u.list = append(u.list, files...)
		u.tree.push(k, files)
	}

	return u, nil
}

// exists if file exists.
func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}

	if os.IsNotExist(err) {
		return false
	}

	return false
}

// pushes new file upload into it's proper place.
func (d fileTree) push(k string, v []*FileUpload) {
	if len(v) == 0 {
		// skip empty values
		return
	}

	indexes := make([]string, 0)
	for _, index := range strings.Split(k, "[") {
		indexes = append(indexes, strings.Trim(index, "]"))
	}

	if len(indexes) <= MaxLevel {
		d.mount(indexes, v)
	}
}

// mount mounts data tree recursively.
func (d fileTree) mount(i []string, v []*FileUpload) {
	if len(v) == 0 {
		return
	}

	if len(i) == 1 {
		// single value context
		d[i[0]] = v[0]
		return
	}

	if len(i) == 2 && i[1] == "" {
		// non associated array of elements
		d[i[0]] = v
		return
	}

	if p, ok := d[i[0]]; ok {
		p.(fileTree).mount(i[1:], v)
	}

	d[i[0]] = make(fileTree)
	d[i[0]].(fileTree).mount(i[1:], v)
}

// fetchIndexes parses input name and splits it into separate indexes list.
func fetchIndexes(s string) []string {
	var (
		pos  int
		ch   string
		keys = make([]string, 1)
	)

	for _, c := range s {
		ch = string(c)
		switch ch {
		case " ":
			// ignore all spaces
			continue
		case "[":
			pos = 1
			continue
		case "]":
			if pos == 1 {
				keys = append(keys, "")
			}
			pos = 2
		default:
			if pos > 0 {
				keys = append(keys, "")
			}

			keys[len(keys)-1] += ch
			pos = 0
		}
	}

	return keys
}
