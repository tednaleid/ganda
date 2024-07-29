package cli

import (
	"fmt"
	"strconv"
)
import "github.com/urfave/cli/v3"

// the urfave/cli package only supports int64 flags, we only want realistic (>0, not too big) values
type WorkerFlag = cli.FlagBase[int, cli.IntegerConfig, intValue]

type intValue struct {
	val  *int
	base int
}

func (i intValue) Create(val int, p *int, c cli.IntegerConfig) cli.Value {
	*p = val
	return &intValue{
		val:  p,
		base: c.Base,
	}
}

func (i intValue) ToString(b int) string {
	return strconv.Itoa(b)
}

func (i *intValue) Set(s string) error {
	v, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	if v < 1 || v > 1<<20 {
		return fmt.Errorf("value out of range: %v is not between 1 and 2^20", v)
	}
	*i.val = v
	return err
}

func (i *intValue) Get() any { return *i.val }

func (i *intValue) String() string { return strconv.Itoa(*i.val) }
