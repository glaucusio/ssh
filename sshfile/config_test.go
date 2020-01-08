package sshfile_test

import (
	"encoding/json"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/glaucusio/ssh/sshfile"
	"github.com/google/go-cmp/cmp"
)

var updateGolden = flag.Bool("update-golden", false, "Update golden files")

func TestHostConfig(t *testing.T) {
	want := &sshfile.HostConfig{
		Port:                  22,
		StrictHostKeyChecking: sshfile.Boolean(true),
		GlobalKnownHostsFile:  "/dev/null",
		UserKnownHostsFile:    "/dev/null",
		TcpKeepAlive:          sshfile.Boolean(true),
		ConnectTimeout:        sshfile.Duration(10 * time.Second),
		ConnectionAttempts:    3,
		ServerAliveInterval:   sshfile.Duration(5 * time.Second),
		ServerAliveCountMax:   10,
	}

	p, err := json.MarshalIndent(want, "", "\t")
	if err != nil {
		t.Fatalf("json.Marshal()=%s", err)
	}

	got := new(sshfile.HostConfig)

	if err := json.Unmarshal(p, got); err != nil {
		t.Fatalf("json.Unmarshal()=%s", err)
	}

	if !cmp.Equal(got, want) {
		t.Fatalf("got != want:\n%s\n", cmp.Diff(got, want))
	}
}

func TestParseConfig(t *testing.T) {
	f, err := os.Open("testdata/config")
	if err != nil {
		t.Fatalf("Open()=%s", err)
	}
	defer f.Close()

	got, err := sshfile.ParseConfig(f)
	if err != nil {
		t.Fatalf("ParseConfig()=%s", err)
	}

	if *updateGolden {
		if err := MarshalFile(got.Hosts(), "testdata/config.golden"); err != nil {
			t.Fatalf("MarshalFile()=%s", err)
		}

		return
	}

	var want sshfile.Hosts

	if err := UnmarshalFile("testdata/config.golden", &want); err != nil {
		t.Fatalf("UnmarshalFile()=%s", err)
	}

	if got := got.Hosts(); !cmp.Equal(got, want) {
		t.Fatalf("got != want:\n%s\n", cmp.Diff(got, want))
	}
}

func TestParseFlags(t *testing.T) {
	var tests [][]string

	if err := UnmarshalFile("testdata/flags.json", &tests); err != nil {
		t.Fatalf("UnmarshalFile()=%s", err)
	}

	var wants []*sshfile.HostConfig

	if !*updateGolden {
		if err := UnmarshalFile("testdata/flags.golden.json", &wants); err != nil {
			t.Fatalf("UnmarshalFile()=%s", err)
		}
	}

	for i, flags := range tests {
		t.Run("", func(t *testing.T) {
			got, err := sshfile.ParseFlags(flags)
			if err != nil {
				t.Fatalf("ParseFlags()=%s", err)
			}

			if *updateGolden {
				wants = append(wants, got)
				return
			}

			if want := wants[i]; !cmp.Equal(got, want) {
				t.Fatalf("got != want:\n%s\n", cmp.Diff(got, want))
			}
		})
	}

	if *updateGolden {
		if err := MarshalFile(wants, "testdata/flags.golden.json"); err != nil {
			t.Fatalf("MarshalFile()=%s", err)
		}
	}
}
