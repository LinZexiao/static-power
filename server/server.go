package server

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"static-power/api"
	"strconv"
	"strings"

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

	srv.GET("/api/v0/miners/csv", func(c *gin.Context) {
		miners, err := a.GetAllMiners()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		// transform to csv
		buf := bytes.NewBuffer([]byte{})
		w := csv.NewWriter(buf)
		w.Write([]string{"miner_id", "peer_id", "multiaddrs", "agent_name", "raw_byte_power", "quality_adj_power"})
		for _, miner := range miners {
			minerID := strconv.Itoa(int(miner.ID))
			peerId := ""
			multiaddrs := ""
			agentName := ""
			rawBytePower := ""
			qualityAdjPower := ""

			if miner.Peer != nil {
				peerId = miner.Peer.PeerId
				if miner.Peer.Multiaddrs != nil {
					multiaddrs = strings.Join(*miner.Peer.Multiaddrs, " ")
				}
			}

			if miner.Agent != nil {
				agentName = miner.Agent.Name
			}

			if miner.Power != nil {
				rawBytePower = miner.Power.RawBytePower.String()
				qualityAdjPower = miner.Power.QualityAdjPower.String()
			}

			w.Write([]string{
				minerID,
				peerId,
				multiaddrs,
				agentName,
				rawBytePower,
				qualityAdjPower,
			})
		}
		w.Flush()

		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment;filename=miners.csv")
		c.Header("Content-Length", strconv.Itoa(buf.Len()))
		c.String(http.StatusOK, buf.String())
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
