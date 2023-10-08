package cli

import (
	"encoding/csv"
	"fmt"
	"os"
	"static-power/core"
	"static-power/util"
	"strconv"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/urfave/cli/v2"
)

var DiffCmd = &cli.Command{
	Name: "diff",
	Flags: []cli.Flag{
		&cli.UintFlag{
			Name:  "skip",
			Usage: "skip the first N lines of csv file",
			Value: 1,
		},
	},
	Usage:     "diff from two csv export by daemon",
	ArgsUsage: `<before.csv> <after.csv>`,
	Action: func(c *cli.Context) error {
		// arg check
		if c.NArg() != 2 {
			return fmt.Errorf("must provide two csv files")
		}

		miners := make([]map[abi.ActorID]core.MinerForDiff, 2)
		QAPs := [][]float64{{0, 0, 0}, {0, 0, 0}}

		for idx, f := range []string{c.Args().Get(0), c.Args().Get(1)} {
			miners[idx] = make(map[abi.ActorID]core.MinerForDiff)

			if _, err := os.Stat(f); os.IsNotExist(err) {
				return fmt.Errorf("file %s does not exist", f)
			}
			file, err := os.Open(f)
			if err != nil {
				return fmt.Errorf("open %s: %w", f, err)
			}
			defer file.Close()
			csvReader := csv.NewReader(file)

			toSkip := c.Uint("skip")
			for {
				if toSkip > 0 {
					toSkip--
					_, err := csvReader.Read()
					if err != nil {
						return fmt.Errorf("skip %s: %w", f, err)
					}
					continue
				}

				row, err := csvReader.Read()
				if err != nil {
					break
				}
				if len(row) != 6 {
					return fmt.Errorf("invalid row %v", row)
				}
				actorNum, err := strconv.ParseUint(row[0], 10, 64)
				if err != nil {
					return fmt.Errorf("invalid actor id %s", row[0])
				}
				actor := abi.ActorID(actorNum)

				defer func() {
					if err := recover(); err != nil {
						fmt.Printf("error occur : %v in raw: %v ", err, row)
						// panic(err)
						return
					}
				}()

				qap := util.PiB(row[5])
				at := core.AgentTypeFromString(row[3])
				if at == core.AgentTypeLotus {
					QAPs[idx][core.AgentTypeLotus] += qap
				} else if at == core.AgentTypeVenus {
					QAPs[idx][core.AgentTypeVenus] += qap
				}

				miners[idx][actor] = core.MinerForDiff{
					Actor: actor,
					Agent: row[3],
					QAP:   qap,
				}
			}
		}

		diffs := core.Diff(miners[0], miners[1])

		// print out diff
		fmt.Printf("Agent, QAP_in_PiB,QAP_diff \n")
		fmt.Printf("venus ,%.5f,%.5f\n", QAPs[1][core.AgentTypeVenus], QAPs[1][core.AgentTypeVenus]-QAPs[0][core.AgentTypeVenus])
		fmt.Printf("lotus ,%.5f,%.5f\n", QAPs[1][core.AgentTypeLotus], QAPs[1][core.AgentTypeLotus]-QAPs[0][core.AgentTypeLotus])
		fmt.Println()

		fmt.Printf("agent,diff_type,actor,qap_change_pib\n")
		for _, diff := range diffs {
			fmt.Printf("%s,%s,%d,%.5f\n", diff.Agent, diff.DiffType, diff.Actor, diff.QAP)
		}

		return nil
	},
}
