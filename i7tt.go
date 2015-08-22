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

// A Linux package to display the CPU temperature of Intel CPUs.
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

const version = "v1.02"

// Here are stored the filenames of the sysfs files we use.
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
	avg_duration  int
	num_of_inputs int
	print_version bool
)

func init() {
	flag.IntVar(&avg_duration, "avg", 30, "avg period in seconds")
	flag.IntVar(&avg_duration, "a", 30, "avg period in seconds"+
		" (shorthand)")
	flag.BoolVar(&print_version, "version", false, "print version")
	flag.BoolVar(&print_version, "v", false, "print version"+" (shorthand)")
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// detect_sensors tries to find sysfs files created from coretemp driver
// that contain the info we seek
func detect_sensors() {
	// Each regexp matches a sysfs file we seek.
	inputs, _ := regexp.Compile("coretemp.*temp([0-9]+)_input")
	labels, _ := regexp.Compile("coretemp.*temp([0-9]+)_label")
	criticals, _ := regexp.Compile("coretemp.*temp([0-9]+)_crit")
	maxes, _ := regexp.Compile("coretemp.*temp([0-9]+)_max")

	// Check populates our filename arrays with matches.
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
// change over time: sensor label, max and critical temperature
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

	if print_version {
		fmt.Println("i7tt", version)
		fmt.Println("https://github.com/andmarios/i7tt")
		os.Exit(0)
	}

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
	bc.Border.Label = " CPU Temperatures (°C), Q to quit"
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

	// temperature holds the current temperatures
	temperature := make([]int, num_of_inputs)
	// temperature_history holds arrays of temperatures history
	temperature_history := make([][]float64, num_of_inputs)
	for i := range temperature_history {
		temperature_history[i] = make([]float64, history_length)
	}
	// temperature_temp_sum holds the current avg period sums.
	// When the periods end we calc the avg and empty the arrays.
	temperature_temp_sum := make([]float64, num_of_inputs)

	// calcset_row_height calculates the height for each row and applies it
	// to our widgets.
	calcset_row_height := func() {
		// Calculate row height
		terminal_height := termui.TermHeight()
		if terminal_height < 4*(num_of_inputs+1)/2 {
			terminal_height = 4 * (num_of_inputs + 1) / 2
		}
		row_height := terminal_height * 2 / (num_of_inputs + 1)
		// Apply height
		bc.Height = row_height
		for i := range lc {
			lc[i].Height = row_height
		}
	}

	// calc_lc_dataoffset calculates the linechart's data offset (slice
	// of data) for the current terminal width.
	// This is needed because the linechart will only show the X first
	// points of data, where X is dependent on width. New datapoints are
	// appended to the end of the data array. Since we dynamically resize
	// the linechart, we have to set dynamically which part of our data
	// arrays may be plotted, then assign this slice to the linecharts'
	// data.
	calc_lc_dataoffset := func() int {
		// Calculate lc data offset
		termui.Body.Width = termui.TermWidth()
		length := (termui.Body.Width/2)*2 - 18
		if length > history_length {
			length = history_length
		}
		return history_length - length
	}

	// set_lc_dataoffset set the data of the linecharts to a slice
	// of the data array, according to pre-calculated offset.
	set_lc_dataoffset := func(offset int) {
		for i := range lc {
			lc[i].Data =
				temperature_history[i][offset:]
		}
	}

	// calcset_bc_barwidth calculate the barchart's barwidth in order
	// for the bars to fill the chart and applies it.
	calcset_bc_barwidth := func() {
		termui.Body.Width = termui.TermWidth()
		bc.BarWidth = ((termui.Body.Width / 2) - 3 - num_of_inputs) /
			num_of_inputs
	}

	// Create a termui grid with our components (2 parts).
	// Part.1/ It is a given that we at least have 2 widgets.
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(6, 0, bc),
			termui.NewCol(6, 0, lc[0])))
	// Part.2/ Add rest of rows dynamically, according to num of sensors.
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

	// Calculate and set the barchart's barwidth from current term width.
	calcset_bc_barwidth()
	// Calculate the linechart's data offset from current term width for
	// future use.
	lc_data_offset := calc_lc_dataoffset()
	// Set initial dataoffset to 0. This causes the X axis to resize well.
	set_lc_dataoffset(0)
	// Calculate and set widget's (row) height
	calcset_row_height()
	// Align and render our grid and widgets.
	termui.Body.Align()
	termui.Render(termui.Body)

	// MainLoop:
	rotate := 0 // used to track the current_avg period current step
	evt := termui.EventCh()
	for {
		select {
		case <-counter:
			// Refresh counter. Read temps and update barchart.
			bc.BarColor = termui.ColorYellow
			// Read temperatures and  update arrays and barchart.
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
			// Check if we need to tick counter_avg
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
			// Apply new data to linecharts.
			set_lc_dataoffset(lc_data_offset)
			termui.Render(termui.Body)
		case e := <-evt:
			// termui event.
			// If q pressed, quit.
			if e.Type == termui.EventKey &&
				(e.Ch == 'q' || e.Ch == 'Q') {
				return
			}
			// If resize event, calculate and apply new barchart
			// barwidth, linechart data offset and widget height.
			if e.Type == termui.EventResize {
				calcset_bc_barwidth()
				lc_data_offset = calc_lc_dataoffset()
				set_lc_dataoffset(lc_data_offset)
				calcset_row_height()
				termui.Body.Align()
				termui.Render(termui.Body)
			}
		}
	}
}
