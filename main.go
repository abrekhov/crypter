package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/abrekhov/crypter/src/crypt"
	mux "github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

var password = "hello"
var uploadPath = "./upload/"
var maxUploadBytes int64 = 100000 * 1024 // MB

/*
x main template
x Upload file
x Download uploaded file on fly
x Encrypt file
x Decrypt file
*/
func main() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
	port, exists := os.LookupEnv("PORT")
	if exists == false {
		port = "80"
	}
	address, exists := os.LookupEnv("ADDRESS")
	if exists == false {
		port = "0.0.0.0"
	}
	r := mux.NewRouter()
	r.Use(loggingMiddleware)
	r.HandleFunc("/", MainHandler)
	r.HandleFunc("/upload", UploadHandler)
	r.HandleFunc("/download/{file}", DownloadHandler)
	// r.Handle("/download/", http.FileServer(http.Dir(uploadPath)))
	r.StrictSlash(true)

	log.Printf("Server started on %s:%s ...\n", address, port)
	err := http.ListenAndServe(address+":"+port, r)
	if err != nil {
		log.Print("Server failed...")
	}
}
func MainHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("views/main.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	t.Execute(w, interface{}(nil))
}

// Stream uploader
// func UploadHandler(w http.ResponseWriter, r *http.Request) {

// 	mpReader, err := r.MultipartReader()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	fileName := ""
// 	for {
// 		p, err := mpReader.NextPart()
// 		if err == io.EOF {
// 			http.Redirect(w, r, "/download?file="+fileName, 302)
// 			return
// 		}
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		fileName = p.FileName()
// 		log.Printf("Uploading file: %#v\n", p.FileName())
// 		slurp, err := ioutil.ReadAll(p)
// 		cryptedSlurp := crypt.Encrypt(slurp, "hello")
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		err = ioutil.WriteFile("./upload/"+p.FileName(), cryptedSlurp, 0664)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 	}
// }

// Only one file uploader
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		log.Print("FILE TOO BIG")
		fmt.Fprint(w, http.ErrContentLength)
		return
	}
	fileType := r.PostFormValue("type")
	log.Print(fileType)
	password := r.PostFormValue("password")
	log.Print("passwd:", password)
	action := r.PostFormValue("action")
	log.Print("action:", action)
	file, fileInfo, err := r.FormFile("uploaded")
	if err != nil {
		log.Print("cant read file")
		fmt.Fprint(w, http.ErrMissingFile)
		return
	}
	defer file.Close()
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Fprint(w, "cant read file bytes")
		return
	}
	var cryptedFileBytes []byte
	if action == "decrypt" {
		cryptedFileBytes, err = crypt.Decrypt(fileBytes, password)
		if err != nil {
			log.Print("Decryption failed")
			fmt.Fprint(w, "Decryption failed")
			return
		}
	} else {
		cryptedFileBytes = crypt.Encrypt(fileBytes, password)
	}
	newFilename := action + "ed" + "_" + fileInfo.Filename
	newPath := filepath.Join(uploadPath, newFilename)
	log.Print("newPath:", newPath)
	newFile, err := os.Create(newPath)
	if err != nil {
		log.Print("cant create file")
		fmt.Fprint(w, "cant create file ")
		return
	}
	defer newFile.Close()
	if _, err := newFile.Write(cryptedFileBytes); err != nil {
		log.Print("cant write file")
		fmt.Fprint(w, "cant write file ")
		return
	}

	http.Redirect(w, r, "/download/"+newFilename, 302)
	return
}

func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	filename := vars["file"]
	filepath := filepath.Join(uploadPath, filename)
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		log.Print("Cant find a file:", err)
		fmt.Fprintf(w, "Cant find a file: %s", err)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	http.ServeFile(w, r, uploadPath+"/"+fileInfo.Name())
	os.Remove(filepath)
	return
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Requested:%s\nUser-agent:%s\n", r.RequestURI, r.UserAgent())
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
