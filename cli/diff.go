package cli

import (
	"encoding/csv"
	"errors"
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

		before, err := readCSV(c.Args().Get(0))
		if err != nil {
			return err
		}
		after, err := readCSV(c.Args().Get(1))
		if err != nil {
			return err
		}
		diffs := core.Diff(before, after)
		sumBefore := core.Summarize(before)
		sumAfter := core.Summarize(after)

		// print out diff
		fmt.Printf("agent, count, count_diff, quality_adj_power ,quality_adj_power_diff \n")
		fmt.Printf("venus , %d , %d ,%.5f,%.5f\n", sumAfter[core.AgentTypeVenus].Count, sumAfter[core.AgentTypeVenus].Count-sumBefore[core.AgentTypeVenus].Count, sumBefore[core.AgentTypeVenus].QAP, sumAfter[core.AgentTypeVenus].QAP-sumBefore[core.AgentTypeVenus].QAP)
		fmt.Printf("lotus , %d , %d ,%.5f,%.5f\n", sumAfter[core.AgentTypeLotus].Count, sumAfter[core.AgentTypeLotus].Count-sumBefore[core.AgentTypeLotus].Count, sumBefore[core.AgentTypeLotus].QAP, sumAfter[core.AgentTypeLotus].QAP-sumBefore[core.AgentTypeLotus].QAP)

		fmt.Println()

		fmt.Printf("agent,diff_type,actor,qap_change_pib\n")
		for _, diff := range diffs {
			fmt.Printf("%s,%s,%d,%.5f\n", diff.Agent, diff.DiffType, diff.Actor, diff.QAP)
		}

		return nil
	},
}

func readCSV(path string) ([]core.MinerBrief, error) {
	ret := make([]core.MinerBrief, 0)

	ret = make([]core.MinerBrief, 0)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("file %s does not exist", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	csvReader := csv.NewReader(file)

	firstLine, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	toSkip := checkCsvVersion(firstLine)
	for {
		if toSkip > 0 {
			toSkip--
			_, err := csvReader.Read()
			if err != nil && !errors.Is(err, csv.ErrFieldCount) {
				return nil, fmt.Errorf("skip %s: %w", path, err)
			}
			continue
		}

		row, err := csvReader.Read()
		if err != nil && !errors.Is(err, csv.ErrFieldCount) {
			break
		}
		// skip empty row
		if len(row) == 0 {
			continue
		}
		if len(row) != 6 {
			return nil, fmt.Errorf("invalid row %v", row)
		}
		actorNum, err := strconv.ParseUint(row[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid actor id %s", row[0])
		}
		actor := abi.ActorID(actorNum)

		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("error occur : %v in raw: %v ", err, row)
				// panic(err)
				return
			}
		}()

		ret = append(ret, core.MinerBrief{
			Actor: actor,
			Agent: row[3],
			QAP:   util.PiB(row[5]),
		})
	}

	return ret, nil
}

func checkCsvVersion(row []string) int {
	if len(row) != 1 {
		return 0
	}
	return core.CsvVersion2Skip(row[0])
}
