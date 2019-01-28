package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/docopt/docopt-go"
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/cog"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/sign-go"
)

var (
	version = "[manual build]"
	usage   = "mpstress " + version + `

Usage:
  mpstress [options] <command>...
  mpstress [options] -w
  mpstress -h | --help
  mpstress --version

Options:
  -u --user <name>   User to authorize on hosts [default: root].
  -w --wait          Just wait for SIGINT (ctrl-c).
  -i --interval <n>  Mpstat <interval> field values. [default: 1]
  -o --output <dir>  Directory to write data. [default: output]
  -h --help          Show this screen.
  --debug            Not implemented.
  --version          Show version.
`
)

var (
	log   *cog.Logger
	debug bool
)

func initLogger(args map[string]interface{}) {
	stderr := lorg.NewLog()
	stderr.SetIndentLines(true)
	stderr.SetFormat(
		lorg.NewFormat("${time} ${level:[%s]:right:short} ${prefix}%s"),
	)

	debug = args["--debug"].(bool)

	if debug {
		stderr.SetLevel(lorg.LevelDebug)
	}

	log = cog.NewLogger(stderr)
}

func main() {
	args, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		panic(err)
	}

	initLogger(args)

	hosts, err := readHosts()
	if err != nil {
		log.Fatal(err)
	}

	waiters, stats, err := runStat(
		hosts,
		args["--user"].(string),
		args["--output"].(string),
		args["--interval"].(string),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof(nil, "mpstat started on %d nodes", len(hosts))

	if args["--wait"].(bool) {
		waitSignal()
	} else {
		err := runCommand(args["<command>"].([]string))
		if err != nil {
			log.Fatalf(err, "unable to run command")
		}
	}

	stats.Process.Signal(syscall.SIGINT)

	waiters.Wait()
}

func waitSignal() {
	sign.Notify(func(signal os.Signal) bool {
		log.Infof(nil, "got SIGINT, terminating process")
		return false
	}, syscall.SIGINT)
}

func runCommand(cmdline []string) error {
	params := []string{}
	if len(cmdline) > 1 {
		params = cmdline[1:]
	}

	cmd := exec.Command(cmdline[0], params...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func readHosts() ([]string, error) {
	contents, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to read stdin",
		)
	}

	data := strings.TrimSpace(string(contents))

	return strings.Split(data, "\n"), nil
}
