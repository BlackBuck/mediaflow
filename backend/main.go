// The main backend
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hibiken/asynq"
)

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// create a global asynq client (address configurable via REDIS_ADDR)
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	defer asynqClient.Close()

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

			// Save the uploaded temp file under uploads with original name (unique with timestamp)
			savedName := fmt.Sprintf("%s_%s", strings.Split(handler.Filename, ".")[0], strings.ReplaceAll(strings.ReplaceAll(tempVideoFile.Name(), "./uploads/", ""), "tmp-", ""))
			finalPath := fmt.Sprintf("./uploads/%s", savedName)
			// close temp file before renaming
			tempVideoFile.Close()
			err = os.Rename(tempVideoFile.Name(), finalPath)
			if err != nil {
				http.Error(w, "Could not save uploaded file", http.StatusInternalServerError)
				return
			}

			// create asynq task payload
			payload, _ := json.Marshal(map[string]string{"input": finalPath, "output": fmt.Sprintf("./processed/%s.wav", strings.Split(handler.Filename, ".")[0])})
			task := asynq.NewTask("task:extract_audio", payload)

			// enqueue the task
			info, err := asynqClient.EnqueueContext(context.Background(), task)
			if err != nil {
				http.Error(w, "Could not enqueue task", http.StatusInternalServerError)
				return
			}

			// respond immediately with task id
			resp := map[string]string{"message": "Your file is being processed", "taskId": info.ID}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
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
