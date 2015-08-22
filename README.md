i7tt - CPU Temperature CLI Monitor [![Build Status](https://travis-ci.org/andmarios/i7tt.svg?branch=master)](https://travis-ci.org/andmarios/i7tt)
=====================

## Introduction

`i7tt` is a simple utility to show the current CPU temperature(s) and a
historical average in the terminal. It uses the Linux sysfs interface of the
[coretemp](https://www.kernel.org/doc/Documentation/hwmon/coretemp) driver,
thus it should support most Intel processors produced after 2005.

<img src="./i7tt.png" alt="i7tt screenshot" type="image/png" width="480">

i7tt stands for _i7 terminal temperature_.

## Usage

To run from source:

    $ git clone https://github.com/andmarios/i7tt
    $ cd i7tt
    $ go get -u
    $ go run i7tt.go

You may also download a [precompiled binary for x86_64](https://github.com/andmarios/i7tt/releases/download/v1.0/i7tt-v1.01-x86_64.tbz).

If you have set your go correctly, you can install it easily:

    $ go get -u github.com/andmarios/i7tt
    $ i7tt

You may set the average period length (default 30 seconds):

    $ i7tt -a 5

## Hacking

Since the sysfs interface of hwmon devices is standard, it should be easy to
add other temperature sensors that expose their information to sysfs via
tweaking the regular expressions that detect the sysfs files. Have a look
to the Linux kernel [hwmon documentation](https://www.kernel.org/doc/Documentation/hwmon/).

Unfortunately I haven't any other hardware to test.
