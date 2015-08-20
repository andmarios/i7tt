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

// A package to display the CPU temperature of Intel CPUs.
package main

import (
	"flag"
	"fmt"
	"github.com/gizak/termui"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Here are stored the filenames of the sys files we use.
var (
	temperature_files []string
	label_files       []string
	critical_files    []string
	max_files         []string
)

// Here are stored the contents of the files described above.
var (
	temperature []int
	label       []string
	critical    []int
	max         []int
)

// Points of history to keep.
var history_length = 500

var (
	avg_duration    int
	num_of_inputs   int
	terminal_height int
)

func init() {
	flag.IntVar(&avg_duration, "avg", 30, "avg period in seconds")
	flag.IntVar(&avg_duration, "a", 30, "avg period in seconds"+
		" (shorthand)")
	flag.IntVar(&terminal_height, "height", 36, "height in rows")
	flag.IntVar(&terminal_height, "h", 36, "height in rows"+" (shorthand)")
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// detect_sensors tries to find files created from coretemp driver
// that contain the info we seek
func detect_sensors() {
	inputs, _ := regexp.Compile("coretemp.*temp([0-9]+)_input")
	labels, _ := regexp.Compile("coretemp.*temp([0-9]+)_label")
	criticals, _ := regexp.Compile("coretemp.*temp([0-9]+)_crit")
	maxes, _ := regexp.Compile("coretemp.*temp([0-9]+)_max")

	check := func(path string, f os.FileInfo, err error) error {
		if inputs.MatchString(path) {
			temperature_files = append(temperature_files, path)
		} else if labels.MatchString(path) {
			label_files = append(label_files, path)
		} else if criticals.MatchString(path) {
			critical_files = append(critical_files, path)
		} else if maxes.MatchString(path) {
			max_files = append(max_files, path)
		}
		return nil
	}

	_ = filepath.Walk("/sys/devices/platform/", check)
}

// read_static_values reads once the contents of files that do not
// change over time: sensor label, sensor max and critical temperature
func read_static_values() {
	// Read temperature labels from /sys
	for _, file := range label_files {
		dat, err := ioutil.ReadFile(file)
		check(err)
		value := strings.TrimSuffix(string(dat), "\n")
		label = append(label, value)
	}

	// Read critical temperatures from /sys
	for _, file := range critical_files {
		dat, err := ioutil.ReadFile(file)
		check(err)
		value_string :=
			strings.TrimSuffix(string(dat), "\n")
		value, err := strconv.Atoi(string(value_string))
		check(err)
		critical = append(critical, value/1000)
	}

	// Read max temperatures from /sys
	for _, file := range max_files {
		dat, err := ioutil.ReadFile(file)
		check(err)
		value_string :=
			strings.TrimSuffix(string(dat), "\n")
		value, err := strconv.Atoi(string(value_string))
		check(err)
		max = append(max, value/1000)
	}
}

func main() {
	flag.Parse()
	detect_sensors()
	read_static_values()

	num_of_inputs = len(temperature_files)
	// You may uncomment the next two lines to test with one less sensor.
	// This is useful to debug for cases of odd and even num of sensors.
	//	num_of_inputs -= 1
	//	temperature_files = temperature_files[1:]
	if num_of_inputs == 0 {
		fmt.Println("No sensors found. Exiting.")
		os.Exit(1)
	}

	// Create a 1 sec refresh counter and a counter based on avg_duration
	// ticks of the 1 sec counter
	counter := time.Tick(1 * time.Second)
	// counter_avg comes from counter. If it was independent, we wouldn't be
	// able to make sure that counter_avg ticks exactly between avg_duration
	// runs of counter
	counter_avg := make(chan int, 5) // Should have at most 1 msg in queue

	// Initialize termui instance
	err := termui.Init()
	check(err)
	defer termui.Close()

	// Create a BarChart
	bc := termui.NewBarChart()
	bc.Border.Label = " CPU Temperatures (°C), Q to quit, ↑↓ to resize"
	bc.Border.LabelFgColor = termui.ColorWhite | termui.AttrBold
	bc.TextColor = termui.ColorMagenta
	bc.DataLabels = label
	bc.NumColor = termui.ColorWhite | termui.AttrBold
	bc.BarGap = 1
	// Set the initial bar max as the critical temp of 1st input minus 10
	bc.SetMax(critical[0] - 10)
	bc.BarColor = termui.ColorRed
	bc.PaddingLeft = 1

	// Create LineCharts
	lc := make([]*termui.LineChart, num_of_inputs)
	for i := range lc {
		lc[i] = termui.NewLineChart()
		lc[i].Border.Label = " " + label[i] + ", " +
			strconv.Itoa(avg_duration) + " sec avg (°C) "
		lc[i].LineColor = termui.ColorMagenta | termui.AttrBold
		lc[i].Border.LabelFgColor = termui.ColorGreen | termui.AttrBold
	}

	// calc_row_height calculates the height for each row and applies it
	// to our widgets.
	calc_row_height := func() {
		// Calculate row height
		row_height := terminal_height * 2 / (num_of_inputs + 1)
		bc.Height = row_height
		for i := range lc {
			lc[i].Height = row_height
		}
	}

	// calc_lc_dataoffset calculates the linechart's data offset (slice
	// of data)
	calc_lc_dataoffset := func() int {
		length := (termui.Body.Width/2)*2 - 18
		if length > history_length {
			length = history_length
		}
		return history_length - length
	}

	// calc_bc_barwidth calculate the barchart's barwidth in order
	// for the bars to fill the chart.
	calc_bc_barwidth := func() int {
		return ((termui.Body.Width / 2) - 3 - num_of_inputs) /
			num_of_inputs
	}

	// temperature holds the current temperatures
	temperature := make([]int, num_of_inputs)
	// temperature_history holds arrays of temperatures history
	temperature_history := make([][]float64, num_of_inputs)
	for i := range temperature_history {
		temperature_history[i] = make([]float64, history_length)
	}
	// temperature_temp_sum holds the current avg period sums
	// when the periods end we calc the avg and empty the arrays
	temperature_temp_sum := make([]float64, num_of_inputs)

	// Create a termui grid with our components.
	// It is a given that we at least have 2 widgets.
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(6, 0, bc),
			termui.NewCol(6, 0, lc[0])))
	// Add rest of rows dynamically, accordinf to number of sensors.
	for i := 1; i < num_of_inputs; i += 2 {
		if num_of_inputs-i > 1 {
			termui.Body.AddRows(
				termui.NewRow(
					termui.NewCol(6, 0, lc[i]),
					termui.NewCol(6, 0, lc[i+1])))
		} else {
			termui.Body.AddRows(
				termui.NewRow(
					termui.NewCol(6, 0, lc[i])))
		}
	}

	calc_row_height()
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
	rotate := 0 // used to track the current_avg period current step
	evt := termui.EventCh()
	for {
		select {
		case <-counter:
			// Refresh counter. Read temps and update barchart.
			bc.BarColor = termui.ColorYellow
			for i, file := range temperature_files {
				dat, err := ioutil.ReadFile(file)
				check(err)
				value_string :=
					strings.TrimSuffix(string(dat), "\n")
				value, err := strconv.Atoi(string(value_string))
				check(err)

				temperature[i] = value / 1000
				temperature_temp_sum[i] += float64(value / 1000)
				if temperature[i] >= max[i] {
					bc.BarColor = termui.ColorRed
				}
			}
			bc.Data = temperature
			rotate++
			if rotate == avg_duration {
				counter_avg <- 1
				rotate = 0
			}
			termui.Render(termui.Body)
		case <-counter_avg:
			// Avg refresh counter. Calculate averages and add them
			// to data.
			for i := range temperature_history {
				temperature_history[i] = append(
					temperature_history[i][1:],
					temperature_temp_sum[i]/
						float64(avg_duration))
				temperature_temp_sum[i] = 0
			}
			for i := range lc {
				lc[i].Data =
					temperature_history[i][lc_dataoffset:]
			}
			termui.Render(termui.Body)
		case e := <-evt:
			// termui event.
			// If q pressed, quit. If arrow up/down resize height.
			if e.Type == termui.EventKey &&
				(e.Ch == 'q' || e.Ch == 'Q') {
				return
			} else if e.Type == termui.EventKey &&
				e.Key == termui.KeyArrowDown {
				terminal_height += (num_of_inputs + 1) / 2
				calc_row_height()
				termui.Body.Align()
				termui.Render(termui.Body)
			} else if e.Type == termui.EventKey &&
				e.Key == termui.KeyArrowUp {
				// We do have a minimum terminal height.
				if terminal_height > 8*(num_of_inputs+1)/2 {
					terminal_height -=
						(num_of_inputs + 1) / 2
					calc_row_height()
					termui.Body.Align()
					termui.Render(termui.Body)
				}
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
