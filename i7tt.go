// A package to display temperature of my i7 laptop.
package main

import (
	"flag"
	_ "fmt"
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

var history_length = 200

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
	label := append([]string(nil), label_files...)
	for i, file := range label_files {
		dat, err := ioutil.ReadFile(file)
		check(err)
		value := strings.TrimSuffix(string(dat), "\n")
		label[i] = value
	}

	counter := time.Tick(1 * time.Second)
	counterH_Tick := avg_duration
	// counterH comes from counter. If it was independent, we wouldn't be
	// able to make sure that counterH ticks always between counterH_Tick
	// runs of counter
	counterH := make(chan int, 1)

	err := termui.Init()
	check(err)
	defer termui.Close()

	// BarChart
	bc := termui.NewBarChart()
	bc.Border.Label = " CPU Temperatures (°C)"
	bc.TextColor = termui.ColorMagenta
	//	bc.Width = 58
	bc.Height = 16
	bc.DataLabels = label
	//	bc.BarWidth = 10
	bc.BarGap = 1
	bc.SetMax(95)
	bc.BarColor = termui.ColorRed
	bc.PaddingLeft = 1

	add_to_label := ", " + strconv.Itoa(counterH_Tick) + " sec avg (°C) "
	// LineChart

	lc := make([]*termui.LineChart, len(temp_files))
	for i := range lc {
		lc[i] = termui.NewLineChart()
		lc[i].Border.Label = " " + label[i] + add_to_label
		lc[i].Height = 16
		lc[i].LineColor = termui.ColorYellow | termui.AttrBold
		lc[i].Border.LabelFgColor = termui.ColorGreen
	}

	temperature := make([]int, len(temp_files))
	temperature_history := make([][]float64, len(temp_files))
	for i := range temperature_history {
		temperature_history[i] = make([]float64, history_length)
	}
	temperature_temp_sum := make([]float64, len(temp_files))

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
	bc.BarWidth = calc_bc_barwidth()
	lc_dataoffset := calc_lc_dataoffset()
	for i := range lc {
		lc[i].Data = temperature_history[i][lc_dataoffset:]
	}

	termui.Render(termui.Body)

	// MainLoop:
	rotate := 0
	for {
		select {
		case <-counter:
			for i, file := range temp_files {
				dat, err := ioutil.ReadFile(file)
				check(err)

				value_string :=
					strings.TrimSuffix(string(dat), "\n")
				value, err := strconv.Atoi(string(value_string))
				check(err)
				// fmt.Println(label[i], "|", value/1000, "°C")
				temperature[i] = value / 1000
				temperature_temp_sum[i] += float64(value / 1000)
			}

			bc.Data = temperature
			rotate++
			if rotate == counterH_Tick {
				counterH <- 1
				rotate = 0
			}
			// termui.Render(bc, lc0, lc1, lc2, lc3, lc4)
			termui.Render(termui.Body)
		case <-counterH:
			for i := range temperature_history {
				temperature_history[i] = append(
					temperature_history[i][1:],
					temperature_temp_sum[i]/float64(counterH_Tick))
			}
			temperature_temp_sum = make([]float64, len(temp_files))
			for i := range lc {
				lc[i].Data = temperature_history[i][lc_dataoffset:]
			}
		case e := <-termui.EventCh():
			// break MainLoop
			if e.Type == termui.EventKey && e.Ch == 'q' {
				return
			}
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

func calc_lc_dataoffset() int {
	length := (termui.Body.Width/2)*2 - 18
	if length > history_length {
		length = history_length
	}
	return history_length - length
}

func calc_bc_barwidth() int {
	return ((termui.Body.Width / 2) - 3 - len(temp_files)) / len(temp_files)
}
