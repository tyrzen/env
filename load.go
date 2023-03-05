package env

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func LoadVars() (err error) {
	env, err := os.Open(".env")
	if err != nil {
		return fmt.Errorf("error during opening environment file: %w", err)
	}

	defer func() {
		if err = env.Close(); err != nil {
			err = fmt.Errorf("error during closing environment file: %w", err)
		}
	}()

	buf := bufio.NewScanner(env)
	buf.Split(bufio.ScanLines)

	for buf.Scan() {
		if keyVal := strings.Split(buf.Text(), "="); len(keyVal) > 1 {
			if err := os.Setenv(keyVal[0], keyVal[1]); err != nil {
				return fmt.Errorf("error during setting environment variable: %w", err)
			}
		}
	}

	return nil
}
