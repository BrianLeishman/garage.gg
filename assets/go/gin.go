package vgg

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/iancoleman/strcase"
)

func handleBindErrors(c *gin.Context, err error) (ok bool) {
	if err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			resp := gin.H{}
			for _, e := range validationErrors {
				resp[strcase.ToLowerCamel(e.Namespace())] = e.Tag()
			}
			c.JSON(http.StatusBadRequest, resp)

			return false
		}
		if e, ok := err.(*json.UnmarshalTypeError); ok {
			if e.Struct != "" || e.Field != "" {
				c.JSON(http.StatusBadRequest, gin.H{strcase.ToLowerCamel(e.Field): fmt.Sprintf("invalid %s", e.Type.String())})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			}

			return false
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}

	return true
}

func BindJSON(c *gin.Context, dest interface{}) (ok bool) {
	return handleBindErrors(c, c.ShouldBindJSON(dest))
}

func BindQuery(c *gin.Context, dest interface{}) (ok bool) {
	return handleBindErrors(c, c.ShouldBindQuery(dest))
}

func BindURI(c *gin.Context, dest interface{}) (ok bool) {
	return handleBindErrors(c, c.ShouldBindUri(dest))
}
