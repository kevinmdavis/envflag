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

package envflag

import (
	"flag"
	"os"
	"strings"
	"testing"
)

func TestBindSuccess(t *testing.T) {
	tests := []struct {
		description string
		env         map[string]string
		cmdLine     []string
		prefixes    []*Prefix
		want        string
	}{
		{
			description: "value from environment",
			prefixes: []*Prefix{AllEnv},
			env: map[string]string{
				"MY_VAR": "env value",
			},
			want: "env value",
		},
		{
			description: "value from command line",
			prefixes: []*Prefix{AllEnv},
			cmdLine:     []string{"--my-var=cmd value"},
			want:        "cmd value",
		},
		{
			description: "value provided on both cmdLine and in env",
			prefixes: []*Prefix{AllEnv},
			env: map[string]string{
				"MY_VAR": "env value",
			},
			cmdLine: []string{"--my-var=cmd value"},
			want:    "cmd value",
		},
		{
			description: "default value",
			prefixes: []*Prefix{AllEnv},
			want:        "default value",
		},
		{
			description: "value from environment with prefix",
			prefixes:    []*Prefix{NewPrefix(""), NewPrefix("MYAPP")},
			env: map[string]string{
				"MYAPP_MY_VAR": "env value",
			},
			want: "env value",
		},
		{
			description: "multiple prefixes",
			prefixes:    []*Prefix{NewPrefix(""), NewPrefix("MYAPP")},
			env: map[string]string{
				"MY_VAR":       "env value 1",
				"MYAPP_MY_VAR": "env value 2",
			},
			// Value from the last prefix defined should win.
			want: "env value 2",
		},
		{
			description: "provided but not defined in non-strict prefix",
			prefixes:    []*Prefix{NewPrefix("MYAPP")},
			env: map[string]string{
				"MYAPP_BAD_NAME": "test123",
			},
			want: "default value",
		},
		{
			description: "environment value with =",
			prefixes:    []*Prefix{NewPrefix("MYAPP")},
			env: map[string]string{
				"MYAPP_MY_VAR": "a=b",
			},
			want: "a=b",
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tc.env {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("Failed to set environment variable %q=%q: %v", k, v, err)
				}
			}
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			testString := fs.String("my-var", "default value", "This is a flag.")
			if err := BindFlagSet(fs, tc.prefixes...); err != nil {
				t.Fatalf("Failed to bind flag set to environment variables: %v", err)
			}
			if err := fs.Parse(tc.cmdLine); err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}
			if *testString != tc.want {
				t.Errorf("--my-var flag value = %q, want %q", *testString, tc.want)
			}
		})
	}
}

func TestBindErrors(t *testing.T) {
	t.Run("called after flag.Parse()", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		_ = fs.Parse(nil)
		if err := BindFlagSet(fs, AllEnv); err == nil {
			t.Error("BindFlagSet() was called after flag.Parse() but did not return an error. Expected an error.")
		}
	})
	t.Run("bad environment variable value", func(t *testing.T) {
		os.Clearenv()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		intVar := fs.Int("int-var", 20, "an int flag.")
		os.Setenv("INT_VAR", "not-a-number")
		err := BindFlagSet(fs, AllEnv)
		if err == nil {
			t.Fatal("Calling BindFlagSet() with INT_VAR=not-a-number did not return an error. Expected an error.")
		}
		want := "invalid value \"not-a-number\" for environment variable INT_VAR"
		if !strings.Contains(err.Error(), want) {
			t.Errorf("BindFlagSet() returned error string %q, want error containing %q", err.Error(), want)
		}
		if *intVar != 20 {
			t.Errorf("BindFlagSet() with invalid environment variable overwrote the default flag value. Got intVar=%d, want default value (20)", *intVar)
		}
	})
	t.Run("environment variable provided but not defined", func(t *testing.T) {
		os.Clearenv()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		os.Setenv("MYAPP_BAD_VAR", "value")
		err := BindFlagSet(fs, NewPrefix("MYAPP", Strict(true)))
		if err == nil {
			t.Fatal("Calling BindFlagSet() with non-existent flag, with a strict prefix, did not return an error. Expected an error.")
		}
		want := "environment variable provided but corresponding flag not defined: MYAPP_BAD_VAR"
		if !strings.Contains(err.Error(), want) {
			t.Errorf("BindFlagSet() returned error string %q, want error containing %q", err.Error(), want)
		}
	})
}

func TestBindUpdatesUsage(t *testing.T) {
	t.Run("single prefix", func(t *testing.T) {
		os.Clearenv()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.String("my-var", "default value", "This is a flag.")
		if err := BindFlagSet(fs, AllEnv); err != nil {
			t.Fatalf("Failed to bind flag set to environment variables: %v", err)
		}
		wantUsage := "This is a flag. [MY_VAR]"
		usage := fs.Lookup("my-var").Usage
		if usage != wantUsage {
			t.Errorf("BindFlagSet() did not update usage string. Got usage = %q, want %q.", usage, wantUsage)
		}
	})
	t.Run("multiple prefixes", func(t *testing.T) {
		os.Clearenv()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.String("my-var", "default value", "This is a flag.")
		if err := BindFlagSet(fs, AllEnv, NewPrefix("MYAPP", Strict(true))); err != nil {
			t.Fatalf("Failed to bind flag set to environment variables: %v", err)
		}
		wantUsage := "This is a flag. [MY_VAR, MYAPP_MY_VAR]"
		usage := fs.Lookup("my-var").Usage
		if usage != wantUsage {
			t.Errorf("BindFlagSet() did not update usage string. Got usage = %q, want %q.", usage, wantUsage)
		}
	})
}

func TestBindErrorHandling(t *testing.T) {
	var exited bool
	exit = func() {
		exited = true
	}
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	_ = fs.Parse(nil)
	_ = BindFlagSet(fs)
	if !exited {
		t.Errorf("BindFlagSet() did not exit after an error when the FlagSet is configured to ExitOnError")
	}
}
