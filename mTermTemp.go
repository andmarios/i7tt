// A package to display temperature of my i7 laptop.
package main

import (
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

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	label := append([]string(nil), label_files...)
	for i, file := range label_files {
		dat, err := ioutil.ReadFile(file)
		check(err)
		value := strings.TrimSuffix(string(dat), "\n")
		label[i] = value
	}

	counter := time.Tick(1 * time.Second)
	counterH_Tick := 30
	counterH := make(chan int, 1) // counterH comes from counter. If it was independent, we wouldn't be able to make sure that counterH ticks always between counterH_Tick runs of counter

	err := termui.Init()
	check(err)
	defer termui.Close()

	// BarChart
	bc := termui.NewBarChart()
	bc.Border.Label = " CPU Temperatures (°C)"
	bc.TextColor = termui.ColorMagenta
	bc.Width = 58
	bc.Height = 16
	bc.DataLabels = label
	bc.BarWidth = 10
	bc.BarGap = 1
	bc.SetMax(95)
	bc.BarColor = termui.ColorRed
	bc.PaddingLeft = 1

	add_to_label := ", " + strconv.Itoa(counterH_Tick) + " sec avg (°C) "
	// LineChart
	lc0 := termui.NewLineChart()
	lc0.Border.Label = " " + label[0] + add_to_label
	lc0.Width = 58
	lc0.Height = 16
	lc0.X = 58 + 1
	lc0.LineColor = termui.ColorCyan | termui.AttrBold
	//	lc0.Mode = "dot"

	lc1 := termui.NewLineChart()
	lc1.Border.Label = " " + label[1] + add_to_label
	lc1.Width = 58
	lc1.Height = 16
	lc1.Y = 16 + 1
	lc1.LineColor = termui.ColorCyan | termui.AttrBold
	//	lc1.Mode = "dot"

	lc2 := termui.NewLineChart()
	lc2.Border.Label = " " + label[2] + add_to_label
	lc2.Width = 58
	lc2.Height = 16
	lc2.X = 58 + 1
	lc2.Y = 16 + 1
	lc2.LineColor = termui.ColorCyan | termui.AttrBold
	//	lc2.Mode = "dot"

	lc3 := termui.NewLineChart()
	lc3.Border.Label = " " + label[3] + add_to_label
	lc3.Width = 58
	lc3.Height = 16
	lc3.Y = 16 + 1 + 16 + 1
	lc3.LineColor = termui.ColorCyan | termui.AttrBold
	//	lc3.Mode = "dot"

	lc4 := termui.NewLineChart()
	lc4.Border.Label = " " + label[4] + add_to_label
	lc4.Width = 58
	lc4.Height = 16
	lc4.X = 58 + 1
	lc4.Y = 16 + 1 + 16 + 1
	lc4.LineColor = termui.ColorCyan | termui.AttrBold
	//	lc4.Mode = "dot"

	temperature := make([]int, len(temp_files))
	temperature_history := make([][]float64, len(temp_files))
	for i := range temperature_history {
		temperature_history[i] = make([]float64, 96)
	}
	temperature_temp_sum := make([]float64, len(temp_files))

	lc0.Data = temperature_history[0]
	lc1.Data = temperature_history[1]
	lc2.Data = temperature_history[2]
	lc3.Data = temperature_history[3]
	lc4.Data = temperature_history[4]

	// MainLoop:
	rotate := 0
	for {
		select {
		case <-counter:
			for i, file := range temp_files {
				dat, err := ioutil.ReadFile(file)
				check(err)

				value_string := strings.TrimSuffix(string(dat), "\n")
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
			termui.Render(bc, lc0, lc1, lc2, lc3, lc4)
		case <-counterH:
			for i := range temperature_history {
				temperature_history[i] = append(temperature_history[i][1:], temperature_temp_sum[i]/float64(counterH_Tick))
			}
			temperature_temp_sum = make([]float64, len(temp_files))
			lc0.Data = temperature_history[0]
			lc1.Data = temperature_history[1]
			lc2.Data = temperature_history[2]
			lc3.Data = temperature_history[3]
			lc4.Data = temperature_history[4]
		case e := <-termui.EventCh():
			// break MainLoop
			if e.Type == termui.EventKey {
				return
			}
		}
	}
}
