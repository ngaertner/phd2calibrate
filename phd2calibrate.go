package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"
)

//import "os"
type Event struct {
	Event string
}

func waitForCalibrateStart(conn io.Reader) {
	var event Event
	scanner := bufio.NewScanner(conn)
	for {
		scanner.Scan()
		json.Unmarshal([]byte(scanner.Text()), &event)
		if event.Event == "StartCalibration" {
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

	// connect to this socket
	conn, err := net.Dial("tcp", "127.0.0.1:4400")
	if err != nil {
		fmt.Print(err.Error())
		return
	}

	fmt.Fprintf(conn, "{\"method\": \"stop_capture\"}\n")
	time.Sleep(1 * time.Second)
	fmt.Fprintf(conn, "{\"method\": \"clear_calibration\"}\n")
	time.Sleep(1 * time.Second)
	fmt.Fprintf(conn, "{\"method\": \"guide\", \"params\": [{\"pixels\": 1.5, \"time\": 8, \"timeout\": 40}, true], \"id\": 42}\n")

	ccalresult := make(chan string)

	waitForCalibrateStart(conn)

	go waitForCalibrateEnd(conn, ccalresult)

	for {
		select {
		case result := <-ccalresult:
			switch result {
			case "timeout":
				println(result)
			case "complete":
				println(result)
			case "failed":
				println(result)
			}
		default:
			continue
		}
		break
	}
	fmt.Fprintf(conn, "{\"method\": \"stop_capture\"}\n")
	time.Sleep(1 * time.Second)
}
