/*
 * Copyright (c) 2019 Nico Gärtner
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 * 1. Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 * 3. Neither the name of mosquitto nor the names of its
 *    contributors may be used to endorse or promote products derived from
 *    this software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

//import "os"
type Event struct {
	Event string
}

type ResponseBool struct {
	Result bool
	Id     int
}

var timeoutLimit int64 = 300

func waitForCalibrateStart(conn io.Reader, ccalstart chan string) {
	var event Event
	scanner := bufio.NewScanner(conn)
	for {
		scanner.Scan()
		json.Unmarshal([]byte(scanner.Text()), &event)
		if event.Event == "StartCalibration" {
			ccalstart <- "start"
			break
		}

	}
}

func waitForCalibrateEnd(conn io.Reader, ccalresult chan string) {
	var event Event
	scanner := bufio.NewScanner(conn)
	for {
		scanner.Scan()
		json.Unmarshal([]byte(scanner.Text()), &event)
		if event.Event == "CalibrationComplete" {
			ccalresult <- "complete"
			break
		} else if event.Event == "CalibrationFailed" {
			ccalresult <- "failed"
		}

	}
}

func waitForBoolResponse(conn io.Reader, id int, cresponse chan string) {
	var response ResponseBool
	scanner := bufio.NewScanner(conn)
	for {
		scanner.Scan()
		json.Unmarshal([]byte(scanner.Text()), &response)
		if response.Id == id {
			cresponse <- strconv.FormatBool(response.Result)
			break
		}
	}
}

func main() {

	flag.Int64Var(&timeoutLimit, "t", 300, "Timeout Limit (seconds)")
	flag.Parse()
	// connect to this socket
	conn, err := net.Dial("tcp", "127.0.0.1:4400")
	if err != nil {
		fmt.Print(err.Error())
		return
	}

	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	go func() {
		sig := <-gracefulStop
		fmt.Printf("caught sig: %+v", sig)
		fmt.Println("")
		fmt.Println("Wait for 1 second to stop calibration")
		fmt.Fprintf(conn, "{\"method\": \"stop_capture\"}\n")
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()

	fmt.Fprintf(conn, "{\"method\": \"get_connected\",\"id\":1}\n")
	cresult_connected := make(chan string)

	go waitForBoolResponse(conn, 1, cresult_connected)

	select {
	case result := <-cresult_connected:
		switch result {
		case "false":
			println("Error - No equipment connected")
			os.Exit(1)
		case "true":
			println("Info - Equipment connected")
		}
	case <-time.After(time.Duration(timeoutLimit) * time.Second):
		println("Timeout - Calibration did not start after " + strconv.FormatInt(timeoutLimit, 10) + " seconds - stopping")
		fmt.Fprintf(conn, "{\"method\": \"stop_capture\"}\n")
		time.Sleep(1 * time.Second)
		os.Exit(1)
	}

	println("Preparation - Stopping capture and resetting calibration")

	fmt.Fprintf(conn, "{\"method\": \"stop_capture\"}\n")
	time.Sleep(1 * time.Second)
	fmt.Fprintf(conn, "{\"method\": \"clear_calibration\"}\n")
	time.Sleep(1 * time.Second)

	println("Start - Autoselecting star and starting calibration")

	fmt.Fprintf(conn, "{\"method\": \"guide\", \"params\": [{\"pixels\": 1.5, \"time\": 8, \"timeout\": 40}, true], \"id\": 42}\n")

	ccalstart := make(chan string)

	go waitForCalibrateStart(conn, ccalstart)

	select {
	case result := <-ccalstart:
		switch result {
		case "start":
			println("Running - Calibration successfully started")
		}
	case <-time.After(time.Duration(timeoutLimit) * time.Second):
		println("Timeout - Calibration did not start after " + strconv.FormatInt(timeoutLimit, 10) + " seconds - stopping")
		fmt.Fprintf(conn, "{\"method\": \"stop_capture\"}\n")
		time.Sleep(1 * time.Second)
		os.Exit(1)
	}

	ccalresult := make(chan string)

	go waitForCalibrateEnd(conn, ccalresult)

	select {
	case result := <-ccalresult:
		switch result {
		case "complete":
			println("Finished - Calibration is complete")
		case "failed":
			println("Error - Calibration failed")
		}
	case <-time.After(time.Duration(timeoutLimit) * time.Second):
		println("Timeout - Calibration did not finish after " + strconv.FormatInt(timeoutLimit, 10) + " seconds - stopping")
		println("Postprocessing - Stopping capture and guiding")
		fmt.Fprintf(conn, "{\"method\": \"stop_capture\"}\n")
		time.Sleep(1 * time.Second)
		os.Exit(1)
	}
	println("Postprocessing - Stopping capture and guiding")
	fmt.Fprintf(conn, "{\"method\": \"stop_capture\"}\n")
	time.Sleep(1 * time.Second)
}
