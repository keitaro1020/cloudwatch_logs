package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/go-playground/validator/v10"
	"log"
	"time"
)

type Param struct {
	AWSProfile string `validate:"required"`
	Filter     string `validate:"required"`
	LogGroup   string `validate:"required"`
	Start      string `validate:"required"`
	End        string `validate:"required"`
}

func (p *Param) StartTime() int64 {
	t, _ := time.ParseInLocation("20060102150405", p.Start, time.Local)
	return t.Unix()
}
func (p *Param) EndTime() int64 {
	t, _ := time.ParseInLocation("20060102150405", p.End, time.Local)
	return t.Unix()
}

func main() {
	param := &Param{}
	flag.StringVar(&param.AWSProfile, "profile", "", "aws profile name")
	flag.StringVar(&param.LogGroup, "log-group", "", "log group name")
	flag.StringVar(&param.Filter, "filter", "", "filter pattern")
	flag.StringVar(&param.Start, "start", "", "log search start")
	flag.StringVar(&param.End, "end", "", "log search end")
	flag.Parse()

	if err := validator.New().Struct(param); err != nil {
		flag.PrintDefaults()
		return
	}

	ctx := context.Background()

	app := NewApp()
	if err := app.Execute(ctx, param); err != nil {
		panic(err)
	}
}

type App struct {
}

func NewApp() *App {
	return &App{}
}

func (a *App) Execute(ctx context.Context, param *Param) error {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(param.AWSProfile),
	)
	if err != nil {
		return err
	}

	log.Printf("param: %#v\n", param)
	client := cloudwatchlogs.NewFromConfig(cfg)

	start := param.StartTime()
	for {
		end := start + 60
		if end > param.EndTime() {
			end = param.EndTime()
		}

		res, err := a.sendQuery(ctx, client, param.Filter, param.LogGroup, start, end)
		if err != nil {
			return nil
		}

		fmt.Printf("Results: %d\n", len(res.Results))

		for _, result := range res.Results {
			for _, field := range result {
				if *field.Field == "@message" {
					fmt.Println(*field.Value)
				}
			}
		}

		start = end
		if start >= param.EndTime() {
			break
		}
	}
	return nil
}

func (a *App) sendQuery(ctx context.Context, client *cloudwatchlogs.Client, filter, logGroup string, start, end int64) (*cloudwatchlogs.GetQueryResultsOutput, error) {
	query := `fields @timestamp, @message, @logStream, @log
| filter @message like /` + filter + `/
| sort @timestamp desc`

	queryInput := &cloudwatchlogs.StartQueryInput{
		LogGroupName: aws.String(logGroup),
		QueryString:  aws.String(query),
		StartTime:    aws.Int64(start),
		EndTime:      aws.Int64(end),
		Limit:        aws.Int32(10000),
	}

	queryOutput, err := client.StartQuery(ctx, queryInput)
	if err != nil {
		return nil, err
	}

	for {
		queryResult, err := client.GetQueryResults(ctx, &cloudwatchlogs.GetQueryResultsInput{
			QueryId: queryOutput.QueryId,
		})
		if err != nil {
			return nil, err
		}
		switch queryResult.Status {
		case types.QueryStatusRunning, types.QueryStatusScheduled:
			time.Sleep(5 * time.Second)
		case types.QueryStatusComplete:
			return queryResult, nil
		default:
			return nil, errors.New("query failed")
		}

	}

}
