//
// Copyright 2015 Marios Andreopoulos
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
//

// A package to display the CPU temperature of my i7 laptop.
package main

import (
	"flag"
	"github.com/gizak/termui"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

var temp_files = []string{
	"/sys/devices/platform/coretemp.0/hwmon/hwmon1/temp1_input",
	"/sys/devices/platform/coretemp.0/hwmon/hwmon1/temp2_input",
	"/sys/devices/platform/coretemp.0/hwmon/hwmon1/temp3_input",
	"/sys/devices/platform/coretemp.0/hwmon/hwmon1/temp4_input",
	"/sys/devices/platform/coretemp.0/hwmon/hwmon1/temp5_input"}

var label_files = []string{
	"/sys/devices/platform/coretemp.0/hwmon/hwmon1/temp1_label",
	"/sys/devices/platform/coretemp.0/hwmon/hwmon1/temp2_label",
	"/sys/devices/platform/coretemp.0/hwmon/hwmon1/temp3_label",
	"/sys/devices/platform/coretemp.0/hwmon/hwmon1/temp4_label",
	"/sys/devices/platform/coretemp.0/hwmon/hwmon1/temp5_label"}

var history_length = 500

var avg_duration int

func init() {
	flag.IntVar(&avg_duration, "avg", 30, "avg period in seconds")
	flag.IntVar(&avg_duration, "a", 30, "avg period in seconds"+
		" (shorthand)")
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	flag.Parse()

	// Read temperature labels from /sys
	label := append([]string(nil), label_files...)
	for i, file := range label_files {
		dat, err := ioutil.ReadFile(file)
		check(err)
		value := strings.TrimSuffix(string(dat), "\n")
		label[i] = value
	}

	// Create a 1 sec refresh counter and a counter based on counterH_Tick
	// ticks of the 1 sec counter
	counter := time.Tick(1 * time.Second)
	counterH_Tick := avg_duration
	// counterH comes from counter. If it was independent, we wouldn't be
	// able to make sure that counterH ticks always between counterH_Tick
	// runs of counter
	counterH := make(chan int, 1)

	// Initialize termui instance
	err := termui.Init()
	check(err)
	defer termui.Close()

	// Create a BarChart
	bc := termui.NewBarChart()
	bc.Border.Label = " CPU Temperatures (°C), press Q to quit"
	bc.Border.LabelFgColor = termui.ColorWhite | termui.AttrBold
	bc.TextColor = termui.ColorMagenta
	bc.Height = 16
	bc.DataLabels = label
	bc.NumColor = termui.ColorWhite | termui.AttrBold
	bc.BarGap = 1
	// I assume a max temp of 95°C. In rare occasions you may see higher temps.
	bc.SetMax(95)
	bc.BarColor = termui.ColorRed
	bc.PaddingLeft = 1

	// This will be added to LineCharts' labels.
	add_to_label := ", " + strconv.Itoa(counterH_Tick) + " sec avg (°C) "
	// Create LineCharts
	lc := make([]*termui.LineChart, len(temp_files))
	for i := range lc {
		lc[i] = termui.NewLineChart()
		lc[i].Border.Label = " " + label[i] + add_to_label
		lc[i].Height = 16
		lc[i].LineColor = termui.ColorMagenta | termui.AttrBold
		lc[i].Border.LabelFgColor = termui.ColorGreen | termui.AttrBold
	}

	// temperature holds the current temperatures
	temperature := make([]int, len(temp_files))
	// temperature_history holds arrays of temperatures history
	temperature_history := make([][]float64, len(temp_files))
	for i := range temperature_history {
		temperature_history[i] = make([]float64, history_length)
	}
	// temperature_temp_sum holds the current avg period sums
	// when the periods end we calc the avg and empty the arrays
	temperature_temp_sum := make([]float64, len(temp_files))

	// Create a termui grid with our components.
	// TODO: Make this part of code dynamic too, based on temperatures
	// found. Every other part I believe is dynamic (except /sys discovery).
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(6, 0, bc),
			termui.NewCol(6, 0, lc[0])),
		termui.NewRow(
			termui.NewCol(6, 0, lc[1]),
			termui.NewCol(6, 0, lc[2])),
		termui.NewRow(
			termui.NewCol(6, 0, lc[3]),
			termui.NewCol(6, 0, lc[4])))
	termui.Body.Align()

	// Calculate the barchart's barwidth from current term width.
	bc.BarWidth = calc_bc_barwidth()
	// Calculate the linechart's data offset from current term width.
	// This is needed because the linechart will only show the X first
	// points of data, where X is dependent on width. New datapoints are
	// appended to the end of the data array. Since we dynamically resize
	// the linechart, we have to set dynamically which part of our data
	// arrays may be plotted, then we assign this slice to the linecharts'
	// data.
	lc_dataoffset := calc_lc_dataoffset()
	for i := range lc {
		lc[i].Data = temperature_history[i][lc_dataoffset:]
	}

	termui.Render(termui.Body)

	// MainLoop:
	rotate := 0 // used to track the current avg period current step
	evt := termui.EventCh()
	for {
		select {
		case <-counter:
			// Refresh counter. Read temps and update barchart.
			for i, file := range temp_files {
				dat, err := ioutil.ReadFile(file)
				check(err)
				value_string :=
					strings.TrimSuffix(string(dat), "\n")
				value, err := strconv.Atoi(string(value_string))
				check(err)
				temperature[i] = value / 1000
				temperature_temp_sum[i] += float64(value / 1000)
			}
			bc.Data = temperature
			rotate++
			if rotate == counterH_Tick {
				counterH <- 1
				rotate = 0
			}
			termui.Render(termui.Body)
		case <-counterH:
			// Avg refresh counter. Calculate averages and add them
			// to data.
			for i := range temperature_history {
				temperature_history[i] = append(
					temperature_history[i][1:],
					temperature_temp_sum[i]/float64(counterH_Tick))
			}
			temperature_temp_sum = make([]float64, len(temp_files))
			for i := range lc {
				lc[i].Data = temperature_history[i][lc_dataoffset:]
			}
			termui.Render(termui.Body)
		case e := <-evt:
			// termui event.
			// If q pressed, quit.
			if e.Type == termui.EventKey && e.Ch == 'q' {
				return
			}
			// If resize event, calculate new barchart barwidth
			// and linechart data offset.
			if e.Type == termui.EventResize {
				termui.Body.Width = termui.TermWidth()
				termui.Body.Align()
				bc.BarWidth = calc_bc_barwidth()
				lc_dataoffset = calc_lc_dataoffset()
				for i := range lc {
					lc[i].Data =
						temperature_history[i][lc_dataoffset:]
				}
				termui.Render(termui.Body)
			}
		}
	}
}

// calc_lc_dataoffset calculates the linechart's data offset (slice of data)
func calc_lc_dataoffset() int {
	length := (termui.Body.Width/2)*2 - 18
	if length > history_length {
		length = history_length
	}
	return history_length - length
}

// calc_bc_barwidth calculate the barchart's barwidth in order for the bars
// to fill the chart.
func calc_bc_barwidth() int {
	return ((termui.Body.Width / 2) - 3 - len(temp_files)) / len(temp_files)
}
