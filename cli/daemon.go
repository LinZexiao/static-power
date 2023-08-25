package cli

import (
	"log"
	"static-power/api"
	sapi "static-power/api"
	"static-power/server"
	"time"

	"github.com/urfave/cli/v2"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var defaultTimeToLive = time.Hour * 24 * 7

var DaemonCmd = &cli.Command{
	Name: "daemon",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "dsn",
			Usage: "database connection string",
		},
		&cli.DurationFlag{
			Name:  "ttl",
			Usage: "time to live for data of db",
		},
	},
	Action: func(c *cli.Context) error {
		var db *gorm.DB
		var err error

		dsn := c.String("dsn")
		listen := c.String("listen")

		if dsn == "" {
			db, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
		} else {
			db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		}
		if err != nil {
			log.Fatal(err)
		}

		a := sapi.NewApi(db)
		go func() {
			// clean db every hour
			var ttl time.Duration
			if c.IsSet("ttl") {
				ttl = c.Duration("ttl")
			} else {
				ttl = defaultTimeToLive
			}

			for {
				api.CleanUp(time.Now().Add(-ttl))
				time.Sleep(time.Hour)
			}
		}()

		server.RegisterApi(a)
		server.Run(listen)
		return nil
	},
}
