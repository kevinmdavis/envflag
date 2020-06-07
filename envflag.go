// Copyright 2020 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package envflag is used to bind environment variables to flag values.
package envflag

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var exit = func() {
	os.Exit(2)
}

// Prefix is a string prefix that's applied to environment variables. For example, the prefix "MYAPP" will map the
// environment variable "MYAPP_PORT" to the flag "port".
type Prefix struct {
	prefix string
	strict bool
}

// NewPrefix creates a prefix with the provided string and strictness. Binding a strict prefix will return an error if
// there are any environment variables matching the prefix that do not have corresponding defined flag.
func NewPrefix(prefix string, opts ...Option) *Prefix {
	p := &Prefix{prefix: prefix}
	for _, o := range opts {
		o(p)
	}
	return p
}

// Option controls how an environment variable prefix is processed.
type Option func(*Prefix)

// Strict ensures that all environment variables with the provided prefix have a corresponding defined flag. If an
// environment variable doesn't have a defined flag, then binding will fail.
func Strict(strict bool) Option {
	return func(o *Prefix) {
		o.strict = strict
	}
}

// NoPrefix matches all environment variables. Unknown environment variables are ignored.
var NoPrefix = NewPrefix("")

func (p Prefix) envName(flagName string) string {
	s := strings.ToUpper(flagName)
	if p.prefix != "" {
		s = fmt.Sprintf("%s_%s", p.prefix, s)
	}
	return strings.ReplaceAll(s, "-", "_")
}

// Bind binds environment variables to the default flag.CommandLine flag set.
func Bind(prefixes ...*Prefix) {
	_ = BindFlagSet(flag.CommandLine, prefixes...)
}

// BindFlagSet binds environment variables to the provided flag set. If there's an error parsing the environment
// variables, the error will be handled using the flag set's configured flag.ErrorHandling.
func BindFlagSet(flagSet *flag.FlagSet, prefixes ...*Prefix) error {
	if err := bind(flagSet, prefixes); err != nil {
		switch flagSet.ErrorHandling() {
		case flag.ContinueOnError:
			return err
		case flag.PanicOnError:
			panic(err)
		default:
			fmt.Fprintln(flagSet.Output(), err)
			if flagSet.Name() == "" {
				fmt.Fprintf(flagSet.Output(), "Usage:\n")
			} else {
				fmt.Fprintf(flagSet.Output(), "Usage of %s:\n", flagSet.Name())
			}
			flagSet.PrintDefaults()
			exit()
		}
	}
	return nil
}

func updateUsage(flag *flag.Flag, prefixes []*Prefix) {
	var envs []string
	for _, p := range prefixes {
		envs = append(envs, p.envName(flag.Name))
	}
	flag.Usage = fmt.Sprintf("%s [%s]", flag.Usage, strings.Join(envs, ", "))
}

func bind(flagSet *flag.FlagSet, prefixes []*Prefix) error {
	if len(prefixes) == 0 {
		prefixes = []*Prefix{NoPrefix}
	}
	if flagSet.Parsed() {
		return fmt.Errorf("envflag.Bind() must be called before flag.Parse()")
	}
	matchedEnvValues := make(map[string]bool)
	var visitErr error
	flagSet.VisitAll(func(f *flag.Flag) {
		updateUsage(f, prefixes)
		for _, p := range prefixes {
			e := p.envName(f.Name)
			if envValue, ok := os.LookupEnv(e); ok {
				matchedEnvValues[e] = true
				prevValue := f.Value.String()
				if err := f.Value.Set(envValue); err != nil {
					visitErr = fmt.Errorf("invalid value %q for environment variable %s: %v", envValue, e, err)
					_ = f.Value.Set(prevValue)
				}
				f.Value.String()
			}
		}
	})
	if visitErr != nil {
		return visitErr
	}
	for _, e := range os.Environ() {
		for _, p := range prefixes {
			if !p.strict {
				continue
			}
			envName := strings.Split(e, "=")[0]
			if strings.HasPrefix(e, p.prefix) && !matchedEnvValues[envName] {
				return fmt.Errorf("environment variable provided but corresponding flag not defined: %s", strings.Split(e, "=")[0])
			}
		}
	}
	return nil
}
