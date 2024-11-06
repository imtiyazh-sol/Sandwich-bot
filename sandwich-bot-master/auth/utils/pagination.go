package utils

import (
	"strconv"

	
	"github.com/gin-gonic/gin"
)

type PaginationType struct {
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Sort   string `json:"sort"`
}

func Pagination(c *gin.Context) PaginationType {
	limit := 1
	offset := 0
	sort := "created_at DESC"
	query := c.Request.URL.Query()

	customSort := map[string]string{}

	for key, value := range query {
		queryValue := value[len(value)-1]
		switch key {
		case "limit":
			limit, _ = strconv.Atoi(queryValue)
			if limit > 50 {
				limit = 50
			}
		case "offset":
			offset, _ = strconv.Atoi(queryValue)
		case "field":
			customSort["field"] = "\"" + queryValue + "\""
		case "sort":
			customSort["sort"] = queryValue
		}
	}

	if len(customSort) == 2 {
		sort = customSort["field"] + " " + customSort["sort"]
	}

	return PaginationType{
		Limit:  limit,
		Offset: offset,
		Sort:   sort,
	}
}
