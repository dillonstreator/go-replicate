package main

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dillonstreator/go-replicate"
)

// https://replicate.com/openai/whisper/versions/e39e354773466b955265e969568deb7da217804d8e771ea8c9cd0cef6591f8bc

type WhisperInput struct {
	// base64 encoded string of audio bytes
	Audio string `json:"audio"`
}

type WhisperOutput struct {
	Segments []struct {
		ID               int     `json:"id"`
		End              float64 `json:"end"`
		Seek             int     `json:"seek"`
		Text             string  `json:"text"`
		Start            float64 `json:"start"`
		Tokens           []int   `json:"tokens"`
		AvgLogprob       float64 `json:"avg_logprob"`
		Temperature      float64 `json:"temperature"`
		NoSpeechProb     float64 `json:"no_speech_prob"`
		CompressionRatio float64 `json:"compression_ratio"`
	} `json:"segments"`
	Translation      *string `json:"translation"`
	Transcription    string  `json:"transcription"`
	DetectedLanguage string  `json:"detected_language"`
}

const whisperModelVersion = "e39e354773466b955265e969568deb7da217804d8e771ea8c9cd0cef6591f8bc"

//go:embed resources/golang.mp3
var audioMP3Bytes []byte

func main() {

	ctx := context.Background()

	replWhisper := replicate.NewClient[WhisperInput, WhisperOutput](os.Getenv("REPLICATE_API_KEY"), whisperModelVersion)

	audio := "data:audio/mp3;base64," + base64.StdEncoding.EncodeToString(audioMP3Bytes)

	prediction, err := replWhisper.CreatePrediction(ctx, WhisperInput{
		Audio: audio,
	})
	if err != nil {
		log.Fatal(err)
	}

	for {
		prediction, err = replWhisper.GetPrediction(ctx, prediction.ID)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(prediction.ID, prediction.Status)
		if prediction.Status == replicate.StatusProcessing || prediction.Status == replicate.StatusStarting {
			time.Sleep(time.Second * 2)
			continue
		}

		fmt.Println(prediction.Output)
		break
	}
}
