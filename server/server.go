package server

import (
	"bytes"
	"encoding/csv"
	"log"
	"net/http"
	"static-power/api"
	"static-power/core"
	"static-power/util"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var srv *gin.Engine = gin.Default()

func RegisterApi(a *api.Api) {
	srv.Use(CORSMiddleware())

	srv.GET("/api/v0/test", func(c *gin.Context) {
		timeParam := c.Query("before")

		before, err := time.Parse(time.RFC3339, timeParam)
		if err != nil {
			log.Printf("parse time params(%s) error: %s\n", timeParam, err.Error())
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message":   "pong",
			"timeParam": timeParam,
			"before":    before,
		})
	})

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
		tag := c.Query("tag")

		p, err := a.GetProportion(api.Option{Tag: tag})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		c.JSON(200, gin.H{"proportion": p})
	})

	srv.GET("/api/v0/static/many/venus", func(c *gin.Context) {
		opt := api.Option{
			Tag:       c.Query("tag"),
			AgentType: core.AgentTypeVenus,
		}

		hours := timeArray()
		ret := make([]*api.StaticInfo, 0, len(hours))

		for _, hour := range hours {
			opt.Before = hour
			s, err := a.GetStatic(opt)
			if err != nil {
				log.Printf("get venus static error: %s\n", err.Error())
				c.JSON(500, gin.H{"error": err.Error()})
			}
			ret = append(ret, s)
		}

		c.JSON(200, ret)
	})

	srv.GET("/api/v0/static/venus", func(c *gin.Context) {
		opt := api.Option{
			Tag:       c.Query("tag"),
			AgentType: core.AgentTypeVenus,
		}

		timeParam := c.Query("before")
		if timeParam != "" {
			var err error
			before, err := time.Parse(time.RFC3339, timeParam)
			if err != nil {
				log.Printf("parse time params(%s) error: %s\n", timeParam, err.Error())
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			opt.Before = before
		}

		s, err := a.GetStatic(opt)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		c.JSON(200, s)
	})

	srv.GET("/api/v0/static/many/lotus", func(c *gin.Context) {
		opt := api.Option{
			Tag:       c.Query("tag"),
			AgentType: core.AgentTypeLotus,
		}

		hours := timeArray()
		ret := make([]*api.StaticInfo, 0, len(hours))

		for _, hour := range hours {
			opt.Before = hour
			s, err := a.GetStatic(opt)
			if err != nil {
				log.Printf("get lotus static error: %s\n", err.Error())
				c.JSON(500, gin.H{"error": err.Error()})
			}
			ret = append(ret, s)
		}

		c.JSON(200, ret)
	})

	srv.GET("/api/v0/static/lotus", func(c *gin.Context) {
		opt := api.Option{
			Tag:       c.Query("tag"),
			AgentType: core.AgentTypeLotus,
		}

		timeParam := c.Query("before")
		if timeParam != "" {
			var err error
			before, err := time.Parse(time.RFC3339, timeParam)
			if err != nil {
				log.Printf("parse time params(%s) error: %s\n", timeParam, err.Error())
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			opt.Before = before
		}

		s, err := a.GetStatic(opt)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		c.JSON(200, s)
	})

	srv.GET("/api/v0/miners/csv", func(c *gin.Context) {
		opt := api.Option{
			Tag: c.Query("tag"),
		}

		timeParam := c.Query("before")
		if timeParam != "" {
			var err error
			before, err := time.Parse(time.RFC3339, timeParam)
			if err != nil {
				log.Printf("parse time params(%s) error: %s\n", timeParam, err.Error())
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			opt.Before = before
		}

		miners, err := a.GetMiners(opt)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}

		// transform to csv
		buf := bytes.NewBuffer([]byte{})
		w := csv.NewWriter(buf)

		const csvVersion = "1.0"
		w.Write([]string{csvVersion})

		// get summary
		// agent count quality_adj_power
		summery := core.Summarize(util.SliceMap(miners, api.GetBrief))
		w.Write([]string{"agent", "count", "quality_adj_power"})
		for k, s := range summery {
			w.Write([]string{
				k.String(),
				strconv.Itoa(s.Count),
				strconv.FormatFloat(s.QAP, 'f', 5, 64),
			})
		}

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

	srv.GET("/api/v0/diff/csv", func(c *gin.Context) {
		opt := api.Option{
			Tag: c.Query("tag"),
		}

		parseTime := func(s string) (time.Time, error) {
			if s == "" {
				return time.Time{}, nil
			}
			return time.Parse(time.RFC3339, s)
		}

		var err error
		opt.Before, err = parseTime(c.Query("before"))
		if err != nil {
			log.Printf("parse time params(%s) error: %s\n", c.Query("before"), err.Error())
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		opt.After, err = parseTime(c.Query("after"))
		if err != nil {
			log.Printf("parse time params(%s) error: %s\n", c.Query("after"), err.Error())
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if opt.Before.IsZero() || opt.After.IsZero() || !opt.Before.Before(opt.After) {
			c.JSON(500, gin.H{"error": "before and after must be set and before should less than after"})
			return
		}

		summaries, difference, err := a.Diff(opt)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}

		// transform to csv
		buf := bytes.NewBuffer([]byte{})
		w := csv.NewWriter(buf)

		const csvVersion = "2.0"
		w.Write([]string{csvVersion})

		// output summary
		// agent count count_diff QAP_in_PiB QAP_diff
		w.Write([]string{"agent", "count", "count_diff", "quality_adj_power", "quality_adj_power_diff"})
		w.Write([]string{
			core.AgentTypeVenus.String(),
			strconv.Itoa(summaries[1][core.AgentTypeVenus].Count),
			strconv.Itoa(summaries[1][core.AgentTypeVenus].Count - summaries[0][core.AgentTypeVenus].Count),
			strconv.FormatFloat(summaries[1][core.AgentTypeVenus].QAP, 'f', 5, 64),
			strconv.FormatFloat(summaries[1][core.AgentTypeVenus].QAP-summaries[0][core.AgentTypeVenus].QAP, 'f', 5, 64),
		})
		w.Write([]string{
			core.AgentTypeLotus.String(),
			strconv.Itoa(summaries[1][core.AgentTypeLotus].Count),
			strconv.Itoa(summaries[1][core.AgentTypeLotus].Count - summaries[0][core.AgentTypeLotus].Count),
			strconv.FormatFloat(summaries[1][core.AgentTypeLotus].QAP, 'f', 5, 64),
			strconv.FormatFloat(summaries[1][core.AgentTypeLotus].QAP-summaries[0][core.AgentTypeLotus].QAP, 'f', 5, 64),
		})

		// agent,diff_type,actor,qap_change_pib
		w.Write([]string{"agent", "diff_type", "actor", "qap_change_pib"})
		for _, d := range difference {
			w.Write([]string{
				d.Agent.String(),
				d.DiffType.String(),
				d.Actor.String(),
				strconv.FormatFloat(d.QAP, 'f', 5, 64),
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

func timeArray() []time.Time {
	ret := make([]time.Time, 0, 7*12+1)

	now := time.Now().Add(time.Hour * 2).Truncate(time.Hour)
	for i := 0; i < 7*24; i++ {
		now = now.Add(-time.Hour)
		if now.Hour()%2 == 0 {
			ret = append(ret, now)
		}
	}
	return ret
}
