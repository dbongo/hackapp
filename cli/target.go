package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/tsuru/tsuru/fs"
	"gopkg.in/v1/yaml"
)

var (
	fsystem        fs.Fs
	TargetFileName = joinHomePath(".backstage_targets")
)

func filesystem() fs.Fs {
	if fsystem == nil {
		fsystem = fs.OsFs{}
	}
	return fsystem
}

type Target struct {
	Current string
	Options map[string]string
}

func (t *Target) GetCommands() []cli.Command {
	return []cli.Command{
		{
			Name:        "target-add",
			Usage:       "target-add <label> <endpoint>",
			Description: "Add a new target in the list of targets.",
			Action: func(c *cli.Context) {
				defer RecoverStrategy("target-add")()
				targets, err := LoadTargets()
				if err != nil {
					fmt.Println(err.Error())
					return
				}
				args := c.Args()
				label, endpoint := args[0], args[1]
				err = targets.add(label, endpoint)
				if err != nil {
					fmt.Println(err.Error())
					return
				}
				fmt.Println("Your new target has been added.")
			},
		},
		{
			Name:        "target-list",
			Usage:       "",
			Description: "Adds a new target in the list of targets.",
			Action: func(c *cli.Context) {
				targets, err := LoadTargets()
				if err != nil {
					fmt.Println(err.Error())
					return
				}
				table := targets.list()
				context := &Context{Stdout: os.Stdout, Stdin: os.Stdin}
				table.Render(context)
			},
		},
		{
			Name:        "target-remove",
			Usage:       "target-remove <label>",
			Description: "Remove a target from the list of targets.",
			Before: func(c *cli.Context) error {
				if c.Args().First() == "" {
					return ErrCommandCancelled
				}
				context := &Context{Stdout: os.Stdout, Stdin: os.Stdin}
				if Confirm(context, "Are you sure you want to remove this target? This action cannot be undone.") != true {
					return ErrCommandCancelled
				}
				return nil
			},
			Action: func(c *cli.Context) {
				defer RecoverStrategy("target-remove")()
				targets, err := LoadTargets()
				if err != nil {
					fmt.Println(err.Error())
					return
				}
				label := c.Args()[1]
				err = targets.remove(label)
				if err != nil {
					fmt.Println(err.Error())
					return
				}
				fmt.Println("The target `" + label + "` has been remove.")
			},
		},
		{
			Name:        "target-set",
			Usage:       "target-set <label>",
			Description: "Set a target as default.",
			Action: func(c *cli.Context) {
				defer RecoverStrategy("target-set")()
				targets, err := LoadTargets()
				if err != nil {
					fmt.Println(err.Error())
					return
				}
				label := c.Args().First()
				err = targets.setDefault(label)
				if err != nil {
					fmt.Println(err.Error())
					return
				}
				fmt.Println("You have a new target as default!")
			},
		},
	}
}

func (t *Target) add(label string, endpoint string) error {
	if _, ok := t.Options[label]; ok {
		return ErrLabelExists
	}
	t.Options[label] = endpoint
	return t.save()
}

func (t *Target) list() *Table {
	table := &Table{
		Content: [][]string{},
		Header:  []string{"Default", "Label", "Backstage Server"},
	}
	sortedKeys := sortMapKeys(t.Options)
	for _, label := range sortedKeys {
		endpoint := t.Options[label]
		line := []string{""}
		if t.Current == label {
			line[0] = "*"
		}
		line = append(line, label, endpoint)
		table.Content = append(table.Content, line)
	}

	return table
}

func (t *Target) remove(label string) error {
	if _, ok := t.Options[label]; !ok {
		return ErrLabelNotFound
	}
	if t.Current == label {
		t.Current = ""
	}
	delete(t.Options, label)
	return t.save()
}

func (t *Target) setDefault(label string) error {
	if _, ok := t.Options[label]; !ok {
		return ErrLabelNotFound
	}
	t.Current = label
	return t.save()
}

func (t *Target) save() error {
	d, err := yaml.Marshal(&t)
	if err != nil {
		return err
	}
	targetsFile, err := filesystem().OpenFile(TargetFileName, syscall.O_RDWR|syscall.O_CREAT|syscall.O_TRUNC, 0600)
	defer targetsFile.Close()
	if err != nil {
		return err
	}
	n, err := targetsFile.WriteString(string(d))
	if n != len(string(d)) || err != nil {
		return ErrFailedWritingTargetFile
	}
	return nil
}

func LoadTargets() (*Target, error) {
	targetsFile, err := filesystem().OpenFile(TargetFileName, syscall.O_RDWR|syscall.O_CREAT, 0600)
	defer targetsFile.Close()
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(targetsFile)
	if err == nil {
		var t Target
		err = yaml.Unmarshal([]byte(data), &t)
		if err != nil {
			return nil, ErrBadFormattedFile
		}
		if t.Options == nil {
			t.Options = map[string]string{}
		}
		return &t, nil
	}
	return nil, err
}
