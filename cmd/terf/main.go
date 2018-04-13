// Copyright 2018 terf Authors. All rights reserved.
//
// This file is part of terf.
//
// terf is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// terf is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with terf.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "terf"
	app.Authors = []cli.Author{cli.Author{Name: "Andrew E. Bruno", Email: "aebruno2@buffalo.edu"}}
	app.Usage = "terf"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		&cli.BoolFlag{Name: "debug,d", Usage: "Print verbose messages"},
	}
	app.Before = func(c *cli.Context) error {
		if c.GlobalBool("debug") {
			log.SetLevel(log.InfoLevel)
		} else {
			log.SetLevel(log.WarnLevel)
		}

		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:  "build",
			Usage: "Converts image data to TFRecords file format with Example protos",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "input,i", Usage: "Path to input file"},
				&cli.StringFlag{Name: "output,o", Usage: "Path to output file"},
				&cli.BoolFlag{Name: "compress,z", Usage: "Use zlib compression"},
			},
			Action: func(c *cli.Context) error {
				err := Build(c.String("input"), c.String("output"), c.Bool("compress"))
				if err != nil {
					log.Fatal(err)
					return cli.NewExitError(err, 1)
				}

				return nil
			},
		},
		{
			Name:  "read",
			Usage: "Read",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "input, i", Usage: "Input file"},
			},
			Action: func(c *cli.Context) error {
				return nil
			},
		}}

	app.RunAndExitOnError()
}
