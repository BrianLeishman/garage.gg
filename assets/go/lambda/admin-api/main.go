package main

import (
	"context"
	"log"
	"os"

	vgg "github.com/BrianLeishman/garage.gg/assets/go"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/guregu/dynamo"

	"github.com/shopspring/decimal"
)

var dev = len(os.Getenv("AWS_EXECUTION_ENV")) == 0

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	decimal.MarshalJSONWithoutQuotes = true
}

var ginLambda *ginadapter.GinLambda

func _init() bool {
	if dev {
		r.Use(cors.New(cors.Config{
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
			AllowHeaders:     []string{"origin", "x-requested-with", "content-type", "accept"},
			ExposeHeaders:    []string{"content-length", "content-type"},
			AllowCredentials: true,
			AllowOriginFunc: func(origin string) bool {
				return true
			},
		}))
	} else {
		// for some reason AWS API Gateway still sends the request to our
		// lambda function even if the request is to check for CORS
		// even though API Gateway handles the CORS...
		r.Use(func(c *gin.Context) {
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}
		})
	}

	return false
}

var r = gin.Default()
var _ = _init()

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	return ginLambda.ProxyWithContext(ctx, req)
}

var table dynamo.Table

func main() {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	db := dynamo.New(sess)
	table = db.Table("vgg")

	binding.Validator = &vgg.Validator{}

	if dev {
		log.Println("running locally http://localhost:8085")
		r.Run(":8085")
	} else {
		gin.SetMode(gin.ReleaseMode)

		ginLambda = ginadapter.New(r)
		lambda.Start(handler)
	}
}
