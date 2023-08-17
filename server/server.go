package server

import (
	"net/http"
	"static-power/api"

	"github.com/gin-gonic/gin"
)

var srv *gin.Engine = gin.Default()

func RegisterApi(a *api.Api) {
	srv.Use(CORSMiddleware())

	srv.GET("/api/v0/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	srv.GET("/api/v0/miner", func(c *gin.Context) {
		miners, err := a.GetAllMiners()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		c.JSON(200, miners)
	})

	srv.GET("/api/v0/proportion", func(c *gin.Context) {
		p, err := a.GetProportion()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		c.JSON(200, gin.H{"proportion": p})
	})

	srv.GET("/api/v0/static/venus", func(c *gin.Context) {
		s, err := a.GetVenusStatic()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		c.JSON(200, s)
	})

	srv.GET("/api/v0/static/lotus", func(c *gin.Context) {
		s, err := a.GetLotusStatic()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		c.JSON(200, s)
	})

	srv.POST("/api/v0/peer", func(c *gin.Context) {
		// get data from body by json
		var peer api.PeerInfo
		c.Bind(&peer)
		err := a.UpdateMinerPeerInfo(&peer)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "ok"})
	})

	srv.POST("/api/v0/agent", func(c *gin.Context) {
		var agent api.AgentInfo
		c.Bind(&agent)
		err := a.UpdateMinerAgentInfo(&agent)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "ok"})
	})

	srv.POST("/api/v0/power", func(c *gin.Context) {
		var power api.PowerInfo
		c.Bind(&power)
		err := a.UpdateMinerPowerInfo(&power)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "ok"})
	})
}

func Run(listen ...string) {
	if len(listen) == 0 {
		listen = append(listen, host)
	}
	srv.Run(listen...)
}

// 跨域请求中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
		}

		c.Next()
	}
}
