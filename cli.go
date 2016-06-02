package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/youyo/emptyport"
)

// Exit codes are int values that represent an exit code for a particular error.
const (
	ExitCodeOK    int = 0
	ExitCodeError int = 1 + iota
)

// CLI is the command line object
type CLI struct {
	// outStream and errStream are the stdout and stderr
	// to write message from the CLI.
	outStream, errStream io.Writer
}

// Config struct
type Config struct {
	Domain   DomainConfig
	Image    ImageConfig
	Template TemplateConfig
}

type DomainConfig struct {
	Name          string
	Vcpu          int
	Memory        int
	Swap          int
	DiskSize      int
	DiskPath      string
	Root_password string
	Arch          string
	Os_type       string
	Os_variant    string
	Network       []NetworkDomain
}

type NetworkDomain struct {
	Iface           string
	Ip              string
	Netmask         string
	Default_gateway string
	Nameserver      string
}

type ImageConfig struct {
	Location string
}

type TemplateConfig struct {
	Server string
	File   string
}

type Output struct {
	Domain   string `json:"domain"`
	Ip       string `json:"ip"`
	Password string `json:"password"`
	Command  string `json:"command"`
}

type OutputErr struct {
	Error string `json:"error"`
}

// Run invokes the CLI with the given arguments.
func (cli *CLI) Run(args []string) int {
	var (
		config  string
		version bool
	)

	// Define option flag parse
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)
	flags.StringVar(&config, "config", "", "Set config file.")
	flags.BoolVar(&version, "version", false, "Print version information and quit.")

	// Parse commandline flag
	if err := flags.Parse(args[1:]); err != nil {
		return ExitCodeError
	}

	// Show version
	if version {
		fmt.Fprintf(cli.errStream, "%s version %s\n", Name, Version)
		return ExitCodeOK
	}

	// Read config
	cfg := readConfig(config)
	// Run http server
	url := cfg.startServer()
	// virt-install
	cmd := cfg.virtInstallCmd(url)
	json := cfg.output()
	v, err := execVirtInstall(cmd)
	if err != nil {
		fmt.Println(v)
	} else {
		fmt.Println(json)
	}

	return ExitCodeOK
}

func readConfig(c string) Config {
	cfg := Config{}
	_, err := toml.DecodeFile(c, &cfg)
	if err != nil {
		panic(err)
	}
	return cfg
}

func (cfg Config) viewHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(cfg.Template.File)
	if err != nil {
		panic(err)
	}

	err = tmpl.Execute(w, cfg)
	if err != nil {
		panic(err)
	}
}

func (cfg Config) startServer() string {
	port := strconv.Itoa(emptyport.Get())
	url := cfg.Template.Server + ":" + port
	http.HandleFunc("/", cfg.viewHandler)
	go http.ListenAndServe(url, nil)
	return url
}

func (cfg Config) virtInstallCmd(serverUrl string) string {
	cmd := "virt-install --debug" +
		" --connect qemu:///system --name " + cfg.Domain.Name +
		" --hvm --virt-type kvm --ram " + strconv.Itoa(cfg.Domain.Memory) +
		" --vcpus " + strconv.Itoa(cfg.Domain.Vcpu) +
		" --arch " + cfg.Domain.Arch +
		" --os-type " + cfg.Domain.Os_type +
		" --os-variant " + cfg.Domain.Os_variant +
		" --disk path=" + cfg.Domain.DiskPath + ",size=" + strconv.Itoa(cfg.Domain.DiskSize) + ",format=qcow2"
	for _, v := range cfg.Domain.Network {
		cmd += " --network " + v.Iface
	}
	cmd += " --graphics vnc,listen=127.0.0.1,keymap=ja --serial pty" +
		" --console pty --noautoconsole --wait -1 --location=" + cfg.Image.Location +
		" --noreboot --autostart --extra-args" +
		" \"ks=http://" + serverUrl +
		" console=tty0 console=ttyS0,115200n8 edd=off keymap=ja" +
		" ip=" + cfg.Domain.Network[0].Ip +
		" netmask=" + cfg.Domain.Network[0].Netmask +
		" gateway=" + cfg.Domain.Network[0].Default_gateway +
		" dns=" + cfg.Domain.Network[0].Nameserver +
		"\""
	return cmd
}

func execVirtInstall(cmd string) (string, error) {
	v, err := exec.Command("bash", "-c", cmd).CombinedOutput()
	if err != nil {
		return string(v), err
	}
	return string(v), nil
}

func (cfg Config) output() string {
	output := Output{
		Domain:   cfg.Domain.Name,
		Ip:       cfg.Domain.Network[0].Ip,
		Password: cfg.Domain.Root_password,
		Command:  "virsh start " + cfg.Domain.Name,
	}
	jsonBytes, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return string(jsonBytes)
}
