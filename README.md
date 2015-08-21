i7tt - CPU Temperature CLI Monitor [![Build Status](https://travis-ci.org/andmarios/i7tt.svg?branch=master)](https://travis-ci.org/andmarios/i7tt)
=====================

## Introduction

`i7tt` is a simple utility to show the current CPU temperature(s) and a historical average in the terminal.
It uses the Linux [coretemp](https://www.kernel.org/doc/Documentation/hwmon/coretemp) driver, thus it should support most Intel processors produced after 2005.

The utility's width is dynamic but due to library limitations the height may only be adjusted by the user, using the arrow keys.

<img src="./i7tt.png" alt="i7tt screenshot" type="image/png" width="480">

i7tt stands for _i7 terminal temperature_.

## Usage

To run from source:

    $ git clone https://github.com/andmarios/i7tt
    $ cd i7tt
    $ go get -u
    $ go run i7tt.go

You may also download a [precompiled binary for x86_64](https://github.com/andmarios/i7tt/releases/download/v1.0rc/i7tt-v1.0rc-x86_64.tbz).

If you have set your go correctly, you can install it easily:

    $ go get -u github.com/andmarios/i7tt
    $ i7tt

You may set the average period length (default 30 seconds):

    $ i7tt -a 5

You may set a starting UI height (default 36 lines):

    $ i7tt -h 48
