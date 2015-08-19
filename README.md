## Introduction

`i7tt` is a simple utility to show the current CPU temperature and a historical average in the terminal.
Out of the box it only works on Linux and with similar CPUs to mine; a four core Intel i7 processor. Check below for adjusting the code to your system.

The utility's width is dynamic but due to library limitations, the height is constant at 48 lines.

<img src="./i7tt.png" alt="i7tt screenshot" type="image/png" width="480">

i7tt stands for _i7 terminal temperature_.

## Usage

To run:

    $ git clone https://github.com/andmarios/i7tt
    $ cd i7tt
    $ go run i7tt.go

If you have set your go correctly, you can install it easily:

    $ go get -u github.com/andmarios/i7tt
    $ i7tt

You may set the average period length (default 30 seconds):

    $ i7tt -a 5

## Hacking

As stateds it only works on similar configurations to mine. It should be fairly easy to adjust for any system though, since most part of the code is dynamic.

To adjust for your system you should make 3 changes:

1. Set your /sys temperature files in `temp_files` variable (at line ~30).
2. Likewise set the /sys label files in `label_files` variable (at line ~37).
3. Adjust the rows and columns of the grid to the number of your temperature inputs + 1.
This happens at line ~126, at the `termui.Body.AddRows()` function. The first widget (1st row, 1st column) is the barchart. The rest are the linecharts. If you have less temperature inputs, remove the unused ones. E.g. if you have three temperatures, remove the whole last row. If you have more, add accordingly.
