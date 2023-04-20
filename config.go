package wzexplorer

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

type wzConfig map[string]string

func parseWzConfig(filename string) (wzConfig, error) {
	fd, err := os.Open(filename + ".ini")
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	config := make(wzConfig)
	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		args := strings.Split(scanner.Text(), "|")
		if len(args) == 0 {
			continue
		}
		if len(args) != 2 {
			return nil, errors.New("invalid config text")
		}
		config[args[0]] = args[1]

	}
	return config, nil
}
