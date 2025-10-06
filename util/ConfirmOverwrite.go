package util

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func ConfirmOverwrite(prompt string, in io.Reader, out io.Writer) (bool, error) {
	fmt.Fprintf(out, "%s (y/N): ", prompt)
	reader := bufio.NewReader(in)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}
