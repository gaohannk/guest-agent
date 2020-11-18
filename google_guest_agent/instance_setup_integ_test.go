//build +integration

package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

/** This only validate when IRQ type is virtionet MSI-X **/
func TestSetMultiQueue(t *testing.T) {
	// Getting necessary information for test
	cmd := exec.Command("nproc")
	ret := runCmdOutput(cmd)
	if ret.ExitCode() != 0 {
		t.Fatalf("got wrong exit code running \"nproc\", expected 0 got %v\n", ret.ExitCode())
	}
	nCpu, err := strconv.Atoi(ret.Stdout()[0 : len(ret.Stdout())-1])
	if err != nil {
		t.Fatalf("can not run integration test without knowning cpu number, %v\n", err)
	}

	// This should be executed when instance boot up, execute again
	if err = setMultiQueue(nCpu); err != nil {
		t.Fatalf("error when setting multiqueue %v\n", err)
	}

	// Validate result: test input and output
	devices, err := filepath.Glob(virtioNetDevs)
	if err != nil {
		t.Fatalf("error when getting virtio device %v\n", err)
	}
	for _, dev := range devices {
		dev = path.Base(dev)
		for _, irq_name := range []string{dev + "-input", dev + "-output"} {
			cmd = exec.Command("bash", "-c", "cat /proc/interrupts | grep "+irq_name+" | awk '{print $1}'| sed 's/://' ")
			ret = runCmdOutput(cmd)
			if ret.ExitCode() != 0 {
				t.Fatalf("got wrong exit code running %s, expected 0 got %v\n", cmd.String(), ret.ExitCode())
			}
			// we only check irq with id in irq_candidate
			irq_candidate := strings.Split(ret.Stdout(), "\n")

			for _, irq := range irq_candidate {
				// read file need to be tested
				bytes, err := ioutil.ReadFile(fmt.Sprint("/proc/irq/%s/smp_affinity_list", irq))
				if err != nil {
					t.Fatalf("Error running test, got %v\n", ret.ExitCode())
				}
				actualSMPAfinitiyList := strings.TrimRight(string(bytes), "\n")
				if actualSMPAfinitiyList != irq {
					t.Errorf("Test failed with expected %s but actual %s.\n", irq, actualSMPAfinitiyList)
				}
			}
		}
	}
	t.Logf(" Test success, the smp_affinity_list setting are correct")
}

func TestConfigureTransmitPacketSteering(t *testing.T) {
	// Getting necessary information for test
	cmd := exec.Command("nproc")
	ret := runCmdOutput(cmd)
	if ret.ExitCode() != 0 {
		t.Fatalf("got wrong exit code running \"nproc\", expected 0 got %v\n", ret.ExitCode())
	}
	nCpu, err := strconv.Atoi(ret.Stdout()[0 : len(ret.Stdout())-1])
	if err != nil {
		t.Fatalf("can not run integration test without knowning cpu number, %v\n", err)
	}
	XPS, err := filepath.Glob(xpsCPU)
	if err != nil {
		t.Fatalf("can not run integration test without knowning XPS, %v\n", err)
	}
	numQueues := len(XPS)
	if nCpu != 48 || numQueues != 32 {
		t.Logf("We only run test for 48 vcpu and 32 queue ")
		return
	}

	// This should be executed when instance boot up, execute again
	if err = configureTransmitPacketSteering(nCpu); err != nil {
		t.Fatalf("error when config XPS %v\n", err)
	}

	// Validate result
	for idx, expectedXPS := range readExpectedResult(t) {
		bytes, err := ioutil.ReadFile(fmt.Sprint("/sys/class/net/eth0/queues/tx-%s/xps_cpus", idx))
		if err != nil {
			t.Fatalf("Error running test, got %v\n", ret.ExitCode())
		}
		actualXPS := strings.TrimRight(string(bytes), "\n")
		if actualXPS != expectedXPS {
			t.Errorf("Test failed with expected %s but actual %s.\n", expectedXPS, actualXPS)
		}
	}

	t.Logf(" Test success, the xps string are correct")
}

func readExpectedResult(t *testing.T) []string {
	file, err := os.Open("expectedXPSStrings.txt")
	if err != nil {
		t.Fatalf("failed opening file: %s", err)
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var expectedXPSStrings []string
	for scanner.Scan() {
		expectedXPSStrings = append(expectedXPSStrings, scanner.Text())
	}
	_ = file.Close()
	return expectedXPSStrings
}
