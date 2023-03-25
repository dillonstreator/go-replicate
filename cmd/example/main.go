package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dillonstreator/go-replicate"
)

type StableDiffusionInput struct {
	Prompt string `json:"prompt"`
}

const stableDiffusionModelVersion = "a9758cbfbd5f3c2094457d996681af52552901775aa2d6dd0b17fd15df959bef"

func main() {

	ctx := context.Background()

	replStableDiffusion := replicate.NewClient[StableDiffusionInput](os.Getenv("REPLICATE_API_KEY"), stableDiffusionModelVersion)

	prediction, err := replStableDiffusion.CreatePrediction(ctx, StableDiffusionInput{
		Prompt: "3d model cat drawn with lines",
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

	iterator := replStableDiffusion.ListPredictions(ctx)
	for {
		item, err := iterator.Next(ctx)
		if err != nil {
			if errors.Is(err, replicate.IteratorDone) {
				break
			}

			log.Fatal(err)
		}

		fmt.Println(item.ID, item.Status)
	}
}
