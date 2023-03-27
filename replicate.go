package replicate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/dillonstreator/request"
	"golang.org/x/exp/slices"
)

type Status string

var IteratorDone = errors.New("iterator done")

const (
	baseURL = "https://api.replicate.com/v1"

	StatusStarting   Status = "starting"
	StatusProcessing Status = "processing"
	StatusSucceeded  Status = "succeeded"
	StatusFailed     Status = "failed"
	StatusCanceled   Status = "canceled"
)

func (s Status) Valid() bool {
	return slices.Contains([]Status{
		StatusStarting,
		StatusProcessing,
		StatusSucceeded,
		StatusFailed,
		StatusCanceled,
	}, s)
}

type Prediction[InputT any, OutputT any] struct {
	ID      string  `json:"id"`
	Version string  `json:"version"`
	Status  Status  `json:"status"`
	Output  OutputT `json:"output"`
	Input   InputT  `json:"input"`
}

type APIError struct {
	Detail string `json:"detail"`
}

func (e APIError) Error() string {
	return e.Detail
}

type Client[InputT any, OutputT any] interface {
	CreatePrediction(ctx context.Context, input InputT) (*Prediction[InputT, OutputT], error)
	GetPrediction(ctx context.Context, id string) (*Prediction[InputT, OutputT], error)
	CancelPrediction(ctx context.Context, id string) error
	ListPredictions(ctx context.Context) Iterator[*PredictionListItem]
}

type client[InputT any, OutputT any] struct {
	requestClient request.Client
	version       string
}

var _ Client[any, any] = (*client[any, any])(nil)

func NewClient[InputT any, OutputT any](apiKey string, modelVersion string) *client[InputT, OutputT] {
	requestClient := request.NewClient(
		baseURL,
		request.WithToken("Token "+apiKey),
		request.WithErrChecker(func(req *http.Request, res *http.Response) error {
			if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
				b, err := io.ReadAll(res.Body)
				if err != nil {
					return err
				}

				apiErr := &APIError{}

				err = json.Unmarshal(b, apiErr)
				if err != nil {
					return err
				}

				return apiErr
			}

			return nil
		}),
	)

	return &client[InputT, OutputT]{
		requestClient: requestClient,
		version:       modelVersion,
	}
}

type input[InputT any] struct {
	Version         string `json:"version"`
	Input           InputT `json:"input"`
	WebhookComplete string `json:"webhook_complete"`
}

func (c *client[InputT, OutputT]) createBody(in InputT) (io.Reader, error) {
	_in := input[InputT]{
		Version: c.version,
		Input:   in,
	}

	body, err := json.Marshal(_in)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(body), nil
}

func (c *client[InputT, OutputT]) CreatePrediction(ctx context.Context, in InputT) (*Prediction[InputT, OutputT], error) {
	body, err := c.createBody(in)
	if err != nil {
		return nil, err
	}

	p := &Prediction[InputT, OutputT]{}

	_, err = c.requestClient.Post(ctx, "/predictions", body, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (c *client[InputT, OutputT]) GetPrediction(ctx context.Context, id string) (*Prediction[InputT, OutputT], error) {
	p := &Prediction[InputT, OutputT]{}

	_, err := c.requestClient.Get(ctx, "/predictions/"+id, nil, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (c *client[InputT, OutputT]) CancelPrediction(ctx context.Context, id string) error {
	_, err := c.requestClient.Post(ctx, "/predictions/"+id+"/cancel", nil, nil)
	if err != nil {
		return err
	}

	return nil
}

type PredictionList struct {
	Results  []*PredictionListItem `json:"results"`
	Next     *string               `json:"next"`
	Previous *string               `json:"previous"`
}

type PredictionListItem struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Status  Status `json:"status"`
}

type Iterator[T any] interface {
	Next(ctx context.Context) (T, error)
}

type predictionListIterator[InputT any, OutputT any] struct {
	client *client[InputT, OutputT]

	currList *PredictionList
	currIdx  int
}

func (p *predictionListIterator[InputT, OutputT]) Next(ctx context.Context) (*PredictionListItem, error) {
	hasNext := p.currList == nil || p.currList.Next != nil
	currDone := p.currList == nil || p.currIdx > len(p.currList.Results)-1

	if currDone {
		if !hasNext {
			return nil, IteratorDone
		}

		var values url.Values
		if p.currList != nil {
			var err error
			values, err = url.ParseQuery(*p.currList.Next)
			if err != nil {
				return nil, err
			}
		}

		currList := &PredictionList{}
		_, err := p.client.requestClient.Get(ctx, "/predictions", values, currList)
		if err != nil {
			return nil, err
		}

		p.currList = currList
		p.currIdx = 0
	}

	next := p.currList.Results[p.currIdx]
	p.currIdx++

	return next, nil
}

func (c *client[InputT, OutputT]) ListPredictions(ctx context.Context) Iterator[*PredictionListItem] {
	return &predictionListIterator[InputT, OutputT]{client: c}
}
