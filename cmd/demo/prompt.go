package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

type choice int

const (
	UP choice = iota
	DOWN
)

func UpDownNoPrompt(contxt context.Context, label string, def choice) chan choice {
	choices := "U/d"
	if def == DOWN {
		choices = "u/D"
	}

	r := bufio.NewReader(os.Stdin)
	var s string

	ch := make(chan choice)

	go func() {
		defer close(ch)
		for {

			select {
			case ch <- func() choice {
				for {
					time.Sleep(1 * time.Second)
					fmt.Fprintf(os.Stderr, "%s (%s) ", label, choices)
					s, _ = r.ReadString('\n')
					s = strings.TrimSpace(s)
					if s == "" {
						return def
					}
					s = strings.ToLower(s)
					if s == "u" || s == "U" || strings.ToUpper(s) == "UP" {
						return UP
					}
					if s == "d" || s == "D" || strings.ToUpper(s) == "DOWN" {
						return DOWN
					}
				}
			}():
			case <-contxt.Done():
				return
			}
		}
	}()
	return ch
}
