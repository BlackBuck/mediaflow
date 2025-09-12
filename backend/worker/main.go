package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hibiken/asynq"
)

type ExtractPayload struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

func handleExtractAudioTask(ctx context.Context, t *asynq.Task) error {
	var p ExtractPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	// ensure output dir exists
	outDir := filepath.Dir(p.Output)
	if err := ensureDir(outDir); err != nil {
		log.Printf("task error: ensureDir failed: %v", err)
		return err
	}

	log.Printf("task start: type=%s input=%s output=%s", t.Type(), p.Input, p.Output)

	// run ffmpeg to convert input to wav
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", p.Input, "-vn", "-acodec", "pcm_s16le", "-ar", "44100", "-ac", "2", p.Output)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("task failed: type=%s err=%v output=%s", t.Type(), err, string(out))
		return fmt.Errorf("ffmpeg failed: %v: %s", err, string(out))
	}

	log.Printf("task complete: type=%s output=%s", t.Type(), p.Output)
	return nil
}

func ensureDir(path string) error {
	if path == "" || path == "." || path == "/" {
		return nil
	}
	return os.MkdirAll(path, 0o755)
}

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc("task:extract_audio", handleExtractAudioTask)

	if err := srv.Run(mux); err != nil {
		panic(err)
	}
}
