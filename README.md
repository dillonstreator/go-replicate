# go-replicate

generic go client for [replicate](https://replicate.com/) http api

## Install

```sh
go get github.com/dillonstreator/go-replicate
```

## Usage

```go
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

const stableDiffusionModelVersion = "a9758cbfbd5f3c2094457d996681af52552901775aa2d6dd0b17fd15df959bef"

type StableDiffusionInput struct {
	Prompt string `json:"prompt"`
}

type StableDiffusionOutput []string

func main() {
	ctx := context.Background()

	stableDiffusion := replicate.NewClient[StableDiffusionInput, StableDiffusionOutput](os.Getenv("REPLICATE_API_KEY"), stableDiffusionModelVersion)

	prediction, err := stableDiffusion.CreatePrediction(ctx, StableDiffusionInput{
		Prompt: "neon sunset into skyline, cyberpunk, tron legacy, grid",
	})
	if err != nil {
		log.Fatal(err)
	}

	for {
		time.Sleep(time.Second * 2)

		prediction, err = stableDiffusion.GetPrediction(ctx, prediction.ID)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(prediction.ID, prediction.Status)
		if prediction.Status == replicate.StatusProcessing {
			continue
		}

		fmt.Println(prediction.Output)
		break
	}
}
```

This example creates a new client that is bound to the [stability-ai/stable-diffusion](https://replicate.com/stability-ai/stable-diffusion/versions/a9758cbfbd5f3c2094457d996681af52552901775aa2d6dd0b17fd15df959bef) model by providing the model input and output structs which gives type safe interactions with the client.

[Explore](https://replicate.com/explore) all available models

More client usage [examples](./examples)
