// The main backend
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	r.Post("/api/upload", func(w http.ResponseWriter, r *http.Request) {
		// Handle file upload and enqueue job
		err := r.ParseMultipartForm(10 << 20) // 10 MB
		if err != nil {
			http.Error(w, "Could not parse multipart form", http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Could not get uploaded file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Now, check if the file's MIME type is video/mp4
		if handler.Header.Get("Content-Type") != "video/mp4" {
			http.Error(w, "Only MP4 video files are allowed", http.StatusBadRequest)
			return
		}

		// if MIME type is video/mp4, run ffmpeg command to convert to audio
		switch handler.Header.Get("Content-Type") {
			case "video/mp4":
				// run ffmpeg command to convert to audio
				tempVideoFile, err := os.CreateTemp("./uploads/", "tmp-*.mp4")
				if err != nil {
					fmt.Println(err)
					http.Error(w, "Could not create temp file", http.StatusInternalServerError)
					return
				}
				defer os.Remove(tempVideoFile.Name())
				defer tempVideoFile.Close()

				_, err = io.Copy(tempVideoFile, file)
				if err != nil {
					http.Error(w, "Could not save temp file", http.StatusInternalServerError)
					return
				}
				
				// Define the output audio file path
				audioFilePath := fmt.Sprintf("./uploads/%s.wav", strings.Split(handler.Filename, ".")[0])
				
				// run ffmpeg without using the shell
				cmd := exec.Command("ffmpeg", "-i", tempVideoFile.Name(), "-vn", "-acodec", "pcm_s16le", "-ar", "44100", "-ac", "2", audioFilePath)
				err = cmd.Run()
				if err != nil {
					http.Error(w, "Could not convert video to audio", http.StatusInternalServerError)
					return
				}
				w.Write([]byte(fmt.Sprintf("File uploaded and converted successfully: %s", audioFilePath)))
				return
			case "audio/wav", "audio/x-wav", "audio/mpeg":
				// temporarily save the uploaded file
				tempAudioFile, err := os.CreateTemp("./uploads/", fmt.Sprintf("tmp-*.%s", strings.Split(handler.Filename, ".")[1]))
				if err != nil {
					http.Error(w, "Could not create temp file", http.StatusInternalServerError)
					return
				}
				defer os.Remove(tempAudioFile.Name())
				defer tempAudioFile.Close()

				_, err = io.Copy(tempAudioFile, file)
				if err != nil {
					http.Error(w, "Could not save temp file", http.StatusInternalServerError)
					return
				}

				// Define the final audio file path
				finalAudioFilePath := fmt.Sprintf("./uploads/%s.wav", strings.Split(handler.Filename, ".")[0])

				// Run ffmpeg command to convert the audio file
				cmd := exec.Command("ffmpeg", "-i", tempAudioFile.Name(), finalAudioFilePath)
				err = cmd.Run()
				if err != nil {
					http.Error(w, "Could not convert audio file", http.StatusInternalServerError)
					return
				}

				w.Write([]byte(fmt.Sprintf("File uploaded and converted successfully: %s", finalAudioFilePath)))
				return
			default:
				http.Error(w, "Unsupported file type", http.StatusBadRequest)
				return
		}

	})

	http.ListenAndServe(":3000", r)
}

