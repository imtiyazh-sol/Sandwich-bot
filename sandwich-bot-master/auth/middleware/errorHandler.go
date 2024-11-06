package middleware

import "github.com/gin-gonic/gin"

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors.Errors()) > 0 {

			var _errorMessage string
			var _data []map[string]interface{}

			for _i, _e := range c.Errors {
				_suffix := "; "
				if _i == 0 {
					_suffix = ""
				}

				_data = append(_data, map[string]interface{}{
					"type":    _e.Type,
					"meta":    _e.Meta,
					"message": _e.Err.Error(),
				})

				_errorMessage += _e.Err.Error() + _suffix
			}
			Response(c, c.Writer.Status(), nil, _errorMessage, "error")
			c.Abort()
		}
	}
}
