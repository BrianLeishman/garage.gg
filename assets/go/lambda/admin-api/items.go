package main

import (
	"bytes"
	"net/http"
	"path/filepath"

	vgg "github.com/BrianLeishman/garage.gg/assets/go"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	"gopkg.in/yaml.v2"
)

func init() {
	r.POST("/items", hItemCreate)
	r.GET("/items", hItemsGet)
	r.GET("/items/:itemID", hItemGet)
	r.DELETE("/items/:itemID", hItemDelete)
}

type Item struct {
	ID          string   `dynamo:"pk" json:"id" yaml:"-" binding:"omitempty,xid"`
	SortKey     string   `dynamo:"sk" json:"-" yaml:"-" binding:"-"`
	Name        string   `dynamo:"data" json:"name" yaml:"title" binding:"required,max=255" mod:"trim"`
	Description string   `dynamo:"desc" json:"description" yaml:"description" binding:"required,max=1024" mod:"trim"`
	Price       float64  `dynamo:"price" json:"price" yaml:"price" binding:"required,gt=0"`
	Sizes       []string `dynamo:"sizes" json:"sizes" yaml:"sizes" binding:"required"`
	Images      []string `dynamo:"images" json:"images" yaml:"images" binding:"required"`
	Body        string   `dynamo:"body" json:"body" yaml:"-" binding:"required" mod:"trim"`
	Draft       bool     `dynamo:"draft" json:"draft" yaml:"draft"`
	Slug        string   `dynamo:"slug" json:"slug" yaml:"slug" binding:"required,max=255" mod:"trim,lcase,tprefix=/"`
	Categories  []string `dynamo:"categories" json:"categories" yaml:"categories" binding:"required" mod:"trim"`
}

func (i *Item) filepath() string {
	return filepath.Join("content", "shop")
}

func (i *Item) filename() string {
	return i.ID + ".md"
}

func (i *Item) markdown() []byte {
	buf := new(bytes.Buffer)
	buf.WriteString(`---
#*****************************************************************************
# This code has been generated by the admin API.
# DO NOT EDIT THIS FILE MANUALLY. All your changes will be lost.
#****************************************************************************/

`)

	yaml.NewEncoder(buf).Encode(i)

	buf.WriteString("\n---\n\n")
	buf.WriteString(i.Body)

	return buf.Bytes()
}

func hItemCreate(c *gin.Context) {
	var req struct {
		Item
	}
	if !vgg.BindJSON(c, &req) {
		return
	}

	req.Item.ID = xid.New().String()
	req.Item.SortKey = "ITEM"

	err := table.Put(req.Item).If("attribute_not_exists('sk')").If("attribute_not_exists('data')").Run()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.Wrapf(err, "failed to insert item"))
		return
	}

	err = commit(&req.Item)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.Wrapf(err, "failed to commit item"))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id": req.Item.ID,
	})
}

func hItemsGet(c *gin.Context) {
	var items []Item
	err := table.Get("sk", "ITEM").Index("sk-data-index").All(&items)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.Wrapf(err, "failed to get items"))
		return
	}

	if items == nil {
		items = make([]Item, 0)
	}

	c.JSON(http.StatusOK, items)
}

func hItemGet(c *gin.Context) {
	var req struct {
		ID string `uri:"itemID" binding:"xid"`
	}
	if !vgg.BindURI(c, &req) {
		return
	}

	var items []Item
	err := table.Get("pk", req.ID).Limit(1).All(&items)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.Wrapf(err, "failed to get item"))
		return
	}

	if len(items) == 0 {
		c.Status(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, items[0])
}

func hItemDelete(c *gin.Context) {
	var req struct {
		ID string `uri:"itemID" binding:"xid"`
	}
	if !vgg.BindURI(c, &req) {
		return
	}

	err := table.Delete("pk", req.ID).Range("sk", "ITEM").Run()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.Wrapf(err, "failed to delete item"))
		return
	}

	err = delete(&Item{ID: req.ID})
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.Wrapf(err, "failed to delete item from repo"))
		return
	}

	c.Status(http.StatusNoContent)
}
