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

type Prediction[InputT any] struct {
	ID      string   `json:"id"`
	Version string   `json:"version"`
	Status  Status   `json:"status"`
	Output  []string `json:"output"`
	Input   InputT   `json:"input"`
}

type APIError struct {
	Detail string `json:"detail"`
}

func (e APIError) Error() string {
	return e.Detail
}

type Client[InputT any] interface {
	CreatePrediction(ctx context.Context, input InputT) (*Prediction[InputT], error)
	GetPrediction(ctx context.Context, id string) (*Prediction[InputT], error)
	ListPredictions(ctx context.Context) Iterator[*PredictionListItem]
}

type client[InputT any] struct {
	requestClient request.Client
	version       string
}

var _ Client[any] = (*client[any])(nil)

func NewClient[InputT any](apiKey string, modelVersion string) *client[InputT] {
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

	return &client[InputT]{
		requestClient: requestClient,
		version:       modelVersion,
	}
}

type input[InputT any] struct {
	Version         string `json:"version"`
	Input           InputT `json:"input"`
	WebhookComplete string `json:"webhook_complete"`
}

func (c *client[InputT]) createBody(in InputT) (io.Reader, error) {
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

func (c *client[InputT]) CreatePrediction(ctx context.Context, in InputT) (*Prediction[InputT], error) {
	body, err := c.createBody(in)
	if err != nil {
		return nil, err
	}

	p := &Prediction[InputT]{}

	_, err = c.requestClient.Post(ctx, "/predictions", body, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (c *client[InputT]) GetPrediction(ctx context.Context, id string) (*Prediction[InputT], error) {
	p := &Prediction[InputT]{}

	_, err := c.requestClient.Get(ctx, "/predictions/"+id, nil, p)
	if err != nil {
		return nil, err
	}

	return p, nil
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

type predictionListIterator[InputT any] struct {
	client *client[InputT]

	currList *PredictionList
	currIdx  int
}

func (p *predictionListIterator[InputT]) Next(ctx context.Context) (*PredictionListItem, error) {
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

func (c *client[InputT]) ListPredictions(ctx context.Context) Iterator[*PredictionListItem] {
	return &predictionListIterator[InputT]{client: c}
}
