package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var env string = "dev"

func If[T any](cond bool, vtrue T, vfalse T) T {
	if cond {
		return vtrue
	}
	return vfalse
}

type Chainstat struct {
	ID         string `json:"id"`
	DiskUsage  int    `json:"disk_usage"`
	EpochFinal int    `json:"epoch_final"`
	Epoch      int    `json:"epoch"`
	EL         struct {
		Status string  `json:"status"`
		Peers  int     `json:"peers"`
		Blk    int     `json:"blk"`
		Bal    float64 `json:"bal"`
	} `json:"el"`
	CL struct {
		Status   string `json:"status"`
		Peers    int    `json:"peers"`
		Syncing  bool   `json:"syncing"`
		Blk      int    `json:"blk"`
		Final    int    `json:"final"`
		Head     int    `json:"head"`
		Expected int    `json:"expected"`
		Dist     int    `json:"dist"`
	} `json:"cl"`
	Val struct {
		Status       string `json:"status"`
		Staking      bool   `json:"staking"`
		TotalCount   int    `json:"total_count"`
		InteropIndex int    `json:"interop_index"`
		InteropCount int    `json:"interop_count"`
	} `json:"val"`
}

func ok(cs Chainstat, minPeers int, maxDist int) bool {
	if cs.DiskUsage < 1 || cs.DiskUsage > 92 {
		// dangerous % disk usage (0 = null/fail)
		// should be treat as unhealthy,
		// even if it's not (to make us aware)
		log.Println("OK check: DiskUsage invalid.")
		return false
	}
	if cs.EL.Status != "enabled/active" ||
		cs.EL.Peers < minPeers ||
		cs.EL.Blk < 1 ||
		(cs.EL.Blk+maxDist) < cs.CL.Blk {
		log.Println("OK check: EL status/peers/blk invalid.")
		return false
	}
	if cs.CL.Status != "enabled/active" ||
		cs.CL.Peers < minPeers ||
		cs.CL.Head < 1 ||
		cs.CL.Expected < 1 ||
		cs.CL.Dist > maxDist {
		log.Println("OK check: CL status/peers/blk/head/expected/dist invalid.")
		return false
	}
	if cs.Val.Status != "enabled/active" &&
		!strings.Contains(cs.Val.Status, "not-found") {
		// only if the validator is installed it must run
		log.Println("OK check: Validator invalid.")
		return false
	}
	return true
}

func main() {
	const mockDataFile = "chainstat-sample.json"

	port := "9788"
	if len(os.Args) >= 2 {
		p, err := strconv.Atoi(os.Args[1])
		if err != nil || p < 1 || p > 65535 {
			log.Fatalf("Invalid port: %q (expected 1..65535)", os.Args[1])
		}
		port = os.Args[1]
	}
	listenAddr := "0.0.0.0:" + port

	minPeers := 3
	if len(os.Args) >= 3 {
		m, err := strconv.Atoi(os.Args[2])
		if err != nil || m < 1 || m > 65535 {
			log.Fatalf("Invalid minPeers: %q (expected 1..65535)", os.Args[2])
		}
		minPeers = m
	}

	maxDist := 3
	if len(os.Args) >= 4 {
		d, err := strconv.Atoi(os.Args[3])
		if err != nil || d < 1 || d > 65535 {
			log.Fatalf("Invalid maxDist: %q (expected 1..65535)", os.Args[3])
		}
		maxDist = d
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var jsonData []byte
		var stderr bytes.Buffer
		if stat, err := os.Stat(mockDataFile); err == nil && !stat.IsDir() &&
			env == "dev" {
			j, _ := os.ReadFile(mockDataFile)
			jsonData = j
		} else {
			cmd := exec.Command("chainstat", "-j")
			cmd.Stderr = &stderr
			j, err := cmd.Output()
			if err != nil {
				log.Printf("chainstat failed: %v; stderr=%q", err, stderr.String())
				http.Error(
					w,
					`{"error":"Failed to run chainstat."}`+"\n",
					http.StatusInternalServerError,
				)
				return
			}
			jsonData = j
		}

		var cs Chainstat
		if err := json.Unmarshal(jsonData, &cs); err != nil {
			log.Printf("unmarshal error: %v; raw=%q; stderr=%q", err, string(jsonData), stderr.String())
			http.Error(
				w,
				`{"error":"Invalid JSON in chainstat data."}`+"\n",
				http.StatusInternalServerError,
			)
			return
		}

		code := If(ok(cs, minPeers, maxDist),
			http.StatusOK,
			http.StatusServiceUnavailable)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(code)
		_, _ = w.Write(jsonData)
		if len(jsonData) == 0 || jsonData[len(jsonData)-1] != '\n' {
			_, _ = w.Write([]byte("\n"))
		}
	})

	log.Println("Environment:", env)
	log.Println("Server starting on", listenAddr)
	log.Println("Required min peers:", minPeers)
	log.Println("Allowed max dist:", maxDist)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
