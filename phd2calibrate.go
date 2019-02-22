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
