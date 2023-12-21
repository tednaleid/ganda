package cli

import "strconv"
import "github.com/urfave/cli/v3"

// the urfave/cli package only supports int64 flags, so we need to add support for int flags
type Int32Flag = cli.FlagBase[int, cli.IntegerConfig, intValue]

type intValue struct {
	val  *int
	base int
}

// Below functions are to satisfy the ValueCreator interface
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

// Below functions are to satisfy the flag.Value interface

func (i *intValue) Set(s string) error {
	v, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	*i.val = v
	return err
}

func (i *intValue) Get() any { return *i.val }

func (i *intValue) String() string { return strconv.Itoa(*i.val) }
