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

/*
Command i7tt shows the temperatures reported by your Intel CPU and
a historical average for each.

    $ go get github.com/andmarios/i7tt
*/
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gizak/termui"
)

const version = "v1.03"

// Here are stored the filenames of the sysfs files we use.
var (
	temperatureFiles []string
	labelFiles       []string
	criticalFiles    []string
	maxFiles         []string
)

// Here are stored the contents of the files described above.
var (
	temperature []int
	label       []string
	critical    []int
	max         []int
)

// Points of history to keep.
var historyLength = 500

var (
	avgDuration  int
	numOfInputs  int
	printVersion bool
)

func init() {
	flag.IntVar(&avgDuration, "avg", 30, "avg period in seconds")
	flag.IntVar(&avgDuration, "a", 30, "avg period in seconds"+
		" (shorthand)")
	flag.BoolVar(&printVersion, "version", false, "print version")
	flag.BoolVar(&printVersion, "v", false, "print version"+" (shorthand)")
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// detect_sensors tries to find sysfs files created from coretemp driver
// that contain the info we seek
func detectSensors() {
	// Each regexp matches a sysfs file we seek.
	inputs, _ := regexp.Compile("coretemp.*temp([0-9]+)_input")
	labels, _ := regexp.Compile("coretemp.*temp([0-9]+)_label")
	criticals, _ := regexp.Compile("coretemp.*temp([0-9]+)_crit")
	maxes, _ := regexp.Compile("coretemp.*temp([0-9]+)_max")

	// Check populates our filename arrays with matches.
	check := func(path string, f os.FileInfo, err error) error {
		if inputs.MatchString(path) {
			temperatureFiles = append(temperatureFiles, path)
		} else if labels.MatchString(path) {
			labelFiles = append(labelFiles, path)
		} else if criticals.MatchString(path) {
			criticalFiles = append(criticalFiles, path)
		} else if maxes.MatchString(path) {
			maxFiles = append(maxFiles, path)
		}
		return nil
	}

	_ = filepath.Walk("/sys/devices/platform/", check)
}

// read_static_values reads once the contents of files that do not
// change over time: sensor label, max and critical temperature
func readStaticValues() {
	// Read temperature labels from /sys
	for _, file := range labelFiles {
		dat, err := ioutil.ReadFile(file)
		check(err)
		value := strings.TrimSuffix(string(dat), "\n")
		label = append(label, value)
	}

	// Read critical temperatures from /sys
	for _, file := range criticalFiles {
		dat, err := ioutil.ReadFile(file)
		check(err)
		valueString :=
			strings.TrimSuffix(string(dat), "\n")
		value, err := strconv.Atoi(string(valueString))
		check(err)
		critical = append(critical, value/1000)
	}

	// Read max temperatures from /sys
	for _, file := range maxFiles {
		dat, err := ioutil.ReadFile(file)
		check(err)
		valueString :=
			strings.TrimSuffix(string(dat), "\n")
		value, err := strconv.Atoi(string(valueString))
		check(err)
		max = append(max, value/1000)
	}
}

func main() {
	flag.Parse()

	if printVersion {
		fmt.Println("i7tt", version)
		fmt.Println("https://github.com/andmarios/i7tt")
		os.Exit(0)
	}

	detectSensors()
	readStaticValues()

	numOfInputs = len(temperatureFiles)
	// You may uncomment the next two lines to test with one less sensor.
	// This is useful to debug for cases of odd and even num of sensors.
	// num_of_inputs -= 1
	// temperature_files = temperature_files[1:]
	if numOfInputs == 0 {
		fmt.Println("No sensors found. Exiting.")
		os.Exit(1)
	}

	// Create a 1 sec refresh counter and a counter based on avg_duration
	// ticks of the 1 sec counter
	counter := time.Tick(1 * time.Second)
	// counter_avg comes from counter. If it was independent, we wouldn't be
	// able to make sure that counter_avg ticks exactly between avg_duration
	// runs of counter
	counterAvg := make(chan int, 5) // Should have at most 1 msg in queue

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
	lc := make([]*termui.LineChart, numOfInputs)
	for i := range lc {
		lc[i] = termui.NewLineChart()
		lc[i].Border.Label = " " + label[i] + ", " +
			strconv.Itoa(avgDuration) + " sec avg (°C) "
		lc[i].LineColor = termui.ColorMagenta | termui.AttrBold
		lc[i].Border.LabelFgColor = termui.ColorGreen | termui.AttrBold
	}

	// temperature holds the current temperatures
	temperature := make([]int, numOfInputs)
	// temperature_history holds arrays of temperatures history
	temperatureHistory := make([][]float64, numOfInputs)
	for i := range temperatureHistory {
		temperatureHistory[i] = make([]float64, historyLength)
	}
	// temperature_temp_sum holds the current avg period sums.
	// When the periods end we calc the avg and empty the arrays.
	temperatureTempSum := make([]float64, numOfInputs)

	// calcset_row_height calculates the height for each row and applies it
	// to our widgets.
	calcsetRowHeight := func() {
		// Calculate row height
		terminalHeight := termui.TermHeight()
		if terminalHeight < 4*(numOfInputs+1)/2 {
			terminalHeight = 4 * (numOfInputs + 2) / 2
		}
		// This equation should stay this way. Since we are dealing with
		// integers which always get rounded down (e.g 1.9 turns 1),
		// the sequence we perform the operations matters.
		// It isn't your normal, real world floating point math. :)
		rowHeight := terminalHeight / ((numOfInputs + 2) / 2)
		// Apply height
		bc.Height = rowHeight
		for i := range lc {
			lc[i].Height = rowHeight
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
	calcLcDataoffset := func() int {
		// Calculate lc data offset
		termui.Body.Width = termui.TermWidth()
		length := (termui.Body.Width/2)*2 - 18
		if length > historyLength {
			length = historyLength
		}
		return historyLength - length
	}

	// set_lc_dataoffset set the data of the linecharts to a slice
	// of the data array, according to pre-calculated offset.
	setLcDataoffset := func(offset int) {
		for i := range lc {
			lc[i].Data =
				temperatureHistory[i][offset:]
		}
	}

	// calcset_bc_barwidth calculate the barchart's barwidth in order
	// for the bars to fill the chart and applies it.
	calcsetBcBarwidth := func() {
		termui.Body.Width = termui.TermWidth()
		bc.BarWidth = ((termui.Body.Width / 2) - 3 - numOfInputs) /
			numOfInputs
	}

	// Create a termui grid with our components (2 parts).
	// Part.1/ It is a given that we at least have 2 widgets.
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(6, 0, bc),
			termui.NewCol(6, 0, lc[0])))
	// Part.2/ Add rest of rows dynamically, according to num of sensors.
	for i := 1; i < numOfInputs; i += 2 {
		if numOfInputs-i > 1 {
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
	calcsetBcBarwidth()
	// Calculate the linechart's data offset from current term width for
	// future use.
	lcDataOffset := calcLcDataoffset()
	// Set initial dataoffset to 0. This causes the X axis to resize well.
	setLcDataoffset(0)
	// Calculate and set widget's (row) height
	calcsetRowHeight()
	// Align and render our grid and widgets.
	termui.Body.Align()
	termui.Render(termui.Body)

	// MainLoop:
	rotate := 0                          // used to track the current_avg period current step
	mediumtemp, hightemp := false, false // used to set barchart color
	evt := termui.EventCh()
	for {
		select {
		case <-counter:
			mediumtemp, hightemp = false, false
			// Read temperatures and update arrays, barchart, color.
			for i, file := range temperatureFiles {
				// Read from sysfs
				dat, err := ioutil.ReadFile(file)
				check(err)
				valueString :=
					strings.TrimSuffix(string(dat), "\n")
				value, err := strconv.Atoi(string(valueString))
				check(err)
				// Update arrays
				temperature[i] = value / 1000
				temperatureTempSum[i] += float64(value / 1000)
				// Update color vars
				// If temperature >= max allowed - 25 °C
				if temperature[i] >= max[i]-25 {
					// if > max allowed => red
					if temperature[i] >= max[i] {
						hightemp = true
					} else { // else yellow
						mediumtemp = true
					}
				}
			}
			bc.Data = temperature
			if hightemp {
				bc.BarColor = termui.ColorRed
			} else if mediumtemp {
				bc.BarColor = termui.ColorYellow
			} else {
				bc.BarColor = termui.ColorGreen
			}
			// Check if we need to tick counter_avg
			rotate++
			if rotate == avgDuration {
				counterAvg <- 1
				rotate = 0
			}
			termui.Render(termui.Body)
		case <-counterAvg:
			// Avg refresh counter. Calculate averages and add them
			// to data.
			for i := range temperatureHistory {
				temperatureHistory[i] = append(
					temperatureHistory[i][1:],
					temperatureTempSum[i]/
						float64(avgDuration))
				temperatureTempSum[i] = 0
			}
			// Apply new data to linecharts.
			setLcDataoffset(lcDataOffset)
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
				calcsetBcBarwidth()
				lcDataOffset = calcLcDataoffset()
				setLcDataoffset(lcDataOffset)
				calcsetRowHeight()
				termui.Body.Align()
				termui.Render(termui.Body)
			}
		}
	}
}
