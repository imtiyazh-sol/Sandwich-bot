package middleware

import (
	"auth/utils"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func Response(c *gin.Context, code int, data any, message string, status string) {
	c.JSON(code, gin.H{
		"status":  status,
		"data":    data,
		"message": message,
	})
}

func DecideErrorMessage(message string, err error) error {
	if message != "" {
		return errors.New(message)
	}
	return err
}

func _Permissions() bool {
	return true
}

func Wrapper(callback func(_data []byte) (int, any, string, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		_data := map[string]interface{}{}
		_, _isInternal := c.Get("internal_request")

		switch c.Request.Method {
		case "PUT", "POST", "PATCH":
			if err := c.ShouldBindJSON(&_data); err != nil {
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}
		default:
			for k, v := range c.Request.URL.Query() {
				trueInt, err := strconv.ParseFloat(v[0], 64)
				if err != nil {
					_data[k] = v[0]
				} else {
					_data[k] = trueInt
				}
			}

			pagination := utils.Pagination(c)
			_data["limit"] = pagination.Limit
			_data["offset"] = pagination.Offset
			_data["sort"] = pagination.Sort
		}

		if _isInternal {
			if !strings.Contains(c.GetHeader("origin"), "bot") {
				c.AbortWithError(http.StatusForbidden, errors.New("Forbidden."))
				return
			}
		} else {
			// Check permissions
			permitted := _Permissions()

			if !permitted {
				c.AbortWithError(http.StatusForbidden, errors.New("Forbidden."))
				return
			}
			// __data, err := json.Marshal(_data)
			// if err != nil {
			// 	log.Println("Error: failed to marshal updated payload: %v", _data)
			// }

			// log request
			// _logger := models.Logger{
			// 	UserID:      int(_data["user_id"].(float64)),
			// 	UserRole:    int(_data["user_role"].(float64)),
			// 	RequestID:   int(_data["request_id"].(float64)),
			// 	RequestPath: c.Request.URL.String(),
			// 	Payload:     __data,
			// 	Method:      c.Request.Method,
			// 	UserAgent:   fmt.Sprintf("%v", _data["user_agent"]),
			// 	ClientIP:    fmt.Sprintf("%v", _data["client_ip"]),
			// }

			// controllers.DB.Model(&models.Logger{}).Create(&_logger)
		}

		payload, err := json.Marshal(_data)

		if err != nil {
			c.AbortWithError(http.StatusUnprocessableEntity, err)
		}

		httpCode, data, message, err := callback(payload)

		if err != nil {
			c.AbortWithError(httpCode, DecideErrorMessage(message, err))
		} else {
			Response(c, httpCode, data, message, "success")
		}
	}
}
