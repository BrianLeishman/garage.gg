package main

import (
	"net/http"

	vgg "github.com/BrianLeishman/garage.gg/assets/go"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/xid"
)

func init() {
	r.POST("/items", hItemCreate)
	// r.PATCH("/users/:user/verify", hUserVerify)
	// r.GET("/users/:user", hUserGet)
}

type Item struct {
	ID          string   `dynamo:"pk" json:"id" binding:"omitempty,startswith=item_,xid,len=25" mod:"trim"`
	Group       string   `dynamo:"sk" json:"group" binding:"required,max=255" mod:"trim"`
	Name        string   `dynamo:"name" json:"name" binding:"required,max=255" mod:"trim"`
	Description string   `dynamo:"desc" json:"description" binding:"required,max=1024" mod:"trim"`
	Price       float64  `dynamo:"price" json:"price" binding:"required,gt=0"`
	Sizes       []string `dynamo:"sizes" json:"sizes" binding:"required"`
	Images      []string `dynamo:"images" json:"images" binding:"required"`
	Body        string   `dynamo:"body" json:"body" binding:"required" mod:"trim"`
	Draft       bool     `dynamo:"draft" json:"draft"`
	Slug        string   `dynamo:"slug" json:"slug" binding:"required,max=255" mod:"trim,lcase,tprefix=/"`
}

func hItemCreate(c *gin.Context) {
	var req struct {
		Item
	}
	if !vgg.BindJSON(c, &req) {
		return
	}

	req.Item.ID = "item_" + xid.New().String()

	err := table.Put(req.Item).Run()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.Wrapf(err, "failed to insert item"))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id": req.Item.ID,
	})
}
