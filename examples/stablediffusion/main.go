package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dillonstreator/go-replicate"
)

// https://replicate.com/stability-ai/stable-diffusion/versions/a9758cbfbd5f3c2094457d996681af52552901775aa2d6dd0b17fd15df959bef

type StableDiffusionInput struct {
	Prompt string `json:"prompt"`
}

type StableDiffusionOutput []string

const stableDiffusionModelVersion = "a9758cbfbd5f3c2094457d996681af52552901775aa2d6dd0b17fd15df959bef"

func main() {

	ctx := context.Background()

	replStableDiffusion := replicate.NewClient[StableDiffusionInput, StableDiffusionOutput](os.Getenv("REPLICATE_API_KEY"), stableDiffusionModelVersion)

	prediction, err := replStableDiffusion.CreatePrediction(ctx, StableDiffusionInput{
		Prompt: "neon sunset into skyline, cyberpunk, tron legacy, grid",
	})
	if err != nil {
		log.Fatal(err)
	}

	for {
		prediction, err = replStableDiffusion.GetPrediction(ctx, prediction.ID)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(prediction.ID, prediction.Status)
		if prediction.Status == replicate.StatusProcessing {
			time.Sleep(time.Second * 2)
			continue
		}

		fmt.Println(prediction.Output)
		break
	}
}
