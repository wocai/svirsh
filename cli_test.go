package main

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestRun_versionFlag(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}
	args := strings.Split("./svirsh -version", " ")

	status := cli.Run(args)
	if status != ExitCodeOK {
		t.Errorf("expected %d to eq %d", status, ExitCodeOK)
	}

	expected := fmt.Sprintf("svirsh version %s", Version)
	if !strings.Contains(errStream.String(), expected) {
		t.Errorf("expected %q to eq %q", errStream.String(), expected)
	}
}

func TestReadConfig(t *testing.T) {
	actual := readConfig("config.toml")
	expected := Config{}
	if reflect.DeepEqual(actual, expected) {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
}

func TestStartServer(t *testing.T) {
	cfg := readConfig("config.toml")
	url := cfg.startServer()
	expected := `192.168.0.9:\d{4,5}`
	m, _ := regexp.MatchString(expected, url)
	if !m {
		t.Errorf("got %v\nwant %v", url, expected)
	}
}
