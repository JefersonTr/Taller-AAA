package middleware

import (
	"log"
	"time"
	"github.com/gin-gonic/gin"
	"automarket/models"
	"automarket/repository"
)

func AccountingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Capturar datos de la petición
		startTime := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()

		// Procesar la petición (ir al endpoint real)
		c.Next()

		// Guardar el log en la persistencia después de ejecutar la operación
		entry := models.LogEntry{
			Timestamp: startTime.Format(time.RFC3339),
			Method:    method,
			Path:      path,
			ClientIP:  clientIP,
		}
		
		repository.DB.SaveLog(entry)

		// Imprimir en consola de forma estructurada con la librería estándar
		log.Printf("[ACCOUNTING] %s | %s | %s | Status: %d\n", 
			entry.Timestamp, method, path, c.Writer.Status())
	}
}