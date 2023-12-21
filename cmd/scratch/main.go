package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/urfave/cli/v3"
	"io"
	"os"
	"strings"
)

func main() {
	cmd := setupCmd(os.Stdin, os.Stdout)
	cmd.Run(context.Background(), os.Args)
}

func setupCmd(in io.Reader, out io.Writer) cli.Command {
	return cli.Command{
		Name:  "rot",
		Usage: "Rotates the input by a specified number of characters",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "rotate",
				Aliases: []string{"r"},
				Value:   13,
				Usage:   "Number of characters to rotate by",
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			rotate := command.Int("rotate")
			return rot(in, out, rotate)
		},
	}
}

func rot(in io.Reader, out io.Writer, rotate int64) error {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		rotated := rotateText(line, rotate)
		fmt.Fprintln(out, rotated)
	}
	return scanner.Err()
}

func rotateText(text string, rotate int64) string {
	var rotated strings.Builder
	for _, r := range text {
		if 'a' <= r && r <= 'z' {
			rotated.WriteRune('a' + (r-'a'+rune(rotate))%26)
		} else if 'A' <= r && r <= 'Z' {
			rotated.WriteRune('A' + (r-'A'+rune(rotate))%26)
		} else {
			rotated.WriteRune(r)
		}
	}
	return rotated.String()
}
