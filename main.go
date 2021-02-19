//go:generate goversioninfo -icon=icon.ico -64=true -manifest=goversioninfo.exe.manifest
//sets the binary properties and icon
// from go get github.com/josephspurrier/goversioninfo/cmd/goversioninfo

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"math/rand"
	net1 "net"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"

	//deal with windows services
	"github.com/kardianos/service"

	//sqllite
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	//stats
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"

	//	"go/types"
	"log"
	"net/http"
	"os/exec"

	//	"strconv"
	//ntfs perms, registry
	"github.com/hectane/go-acl"
	_ "github.com/hectane/go-acl"
	"golang.org/x/sys/windows"
	_ "golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
)

//Version = version number
const Version = "1.0.0.1"

//DiskStats structure for diskstats return
type DiskStats struct {
	//gorm.Model
	ID                uint      `json:"ID"`
	CreatedAt         time.Time `json:"CreatedAt"`
	Path              string    `json:"path"`
	Fstype            string    `json:"fstype"`
	Total             uint64    `json:"total"`
	Free              uint64    `json:"free"`
	Used              uint64    `json:"used"`
	UsedPercent       float64   `json:"usedPercent"`
	InodesTotal       uint64    `json:"inodesTotal"`
	InodesUsed        uint64    `json:"inodesUsed"`
	InodesFree        uint64    `json:"inodesFree"`
	InodesUsedPercent float64   `json:"inodesUsedPercent"`
}

// MemStats is for memory stats returns
type MemStats struct {
	Total          int64   `json:"total"`
	Available      int64   `json:"available"`
	Used           int64   `json:"used"`
	UsedPercent    float64 `json:"usedPercent"`
	Free           int64   `json:"free"`
	Active         int64   `json:"active"`
	Inactive       int64   `json:"inactive"`
	Wired          int64   `json:"wired"`
	Laundry        int     `json:"laundry"`
	Buffers        int     `json:"buffers"`
	Cached         int     `json:"cached"`
	Writeback      int     `json:"writeback"`
	Dirty          int     `json:"dirty"`
	Writebacktmp   int     `json:"writebacktmp"`
	Shared         int     `json:"shared"`
	Slab           int     `json:"slab"`
	Sreclaimable   int     `json:"sreclaimable"`
	Sunreclaim     int     `json:"sunreclaim"`
	Pagetables     int     `json:"pagetables"`
	Swapcached     int     `json:"swapcached"`
	Commitlimit    int     `json:"commitlimit"`
	Committedas    int     `json:"committedas"`
	Hightotal      int     `json:"hightotal"`
	Highfree       int     `json:"highfree"`
	Lowtotal       int     `json:"lowtotal"`
	Lowfree        int     `json:"lowfree"`
	Swaptotal      int     `json:"swaptotal"`
	Swapfree       int     `json:"swapfree"`
	Mapped         int     `json:"mapped"`
	Vmalloctotal   int     `json:"vmalloctotal"`
	Vmallocused    int     `json:"vmallocused"`
	Vmallocchunk   int     `json:"vmallocchunk"`
	Hugepagestotal int     `json:"hugepagestotal"`
	Hugepagesfree  int     `json:"hugepagesfree"`
	Hugepagesize   int     `json:"hugepagesize"`
}

// MemStats1 is for memory stats returns
type MemStats1 struct {
	ID          uint      `json:"ID"`
	CreatedAt   time.Time `json:"CreatedAt"`
	Total       uint64    `json:"total"`
	Available   uint64    `json:"available"`
	Used        uint64    `json:"used"`
	UsedPercent float64   `json:"usedPercent"`
	Free        uint64    `json:"free"`
	Active      uint64    `json:"active"`
	Inactive    uint64    `json:"inactive"`
}

//Diskio structure for Diskio return
type Diskio struct {
	//	gorm.Model
	ID               uint      `json:"ID"`
	CreatedAt        time.Time `json:"CreatedAt"`
	ReadCount        uint64    `json:"readCount"`
	MergedReadCount  uint64    `json:"mergedReadCount"`
	WriteCount       uint64    `json:"writeCount"`
	MergedWriteCount uint64    `json:"mergedWriteCount"`
	ReadBytes        uint64    `json:"readBytes"`
	WriteBytes       uint64    `json:"writeBytes"`
	ReadTime         uint64    `json:"readTime"`
	WriteTime        uint64    `json:"writeTime"`
	IopsInProgress   uint64    `json:"iopsInProgress"`
	IoTime           uint64    `json:"ioTime"`
	WeightedIO       uint64    `json:"weightedIO"`
	Name             string    `json:"name"`
	SerialNumber     string    `json:"serialNumber"`
	Label            string    `json:"label"`
}

//CPUStats struct only has 1 field
type CPUStats struct {
	ID        uint      `json:"ID"`
	CreatedAt time.Time `json:"CreatedAt"`
	Average   int       `json:"avg"`
}

//ServerStats puts some of it together
type ServerStats struct {
	UsedPercentMem  float64 `json:"usedPercentMem"`
	UsedPercentDisk float64 `json:"usedPercentDisk"`
	AverageCPU      int     `json:"avgCpu"`
	IP              string  `json:"ip"`
	ServerName      string  `json:"serverName"`
}

//ExportsJSON shows the current list of nfs exported disks, empty json object if none {}
type ExportsJSON struct {
	Anonymousaccess bool   `json:"anonymousaccess"`
	Anonymousgid    int64  `json:"anonymousgid"`
	Anonymousuid    int64  `json:"anonymousuid"`
	Isonline        bool   `json:"isonline"`
	Name            string `json:"name"`
	Path            string `json:"path"`
}

//NetworkIoStats holds structure for networkiostats
type NetworkIoStats struct {
	ID          uint      `json:"ID"`
	CreatedAt   time.Time `json:"CreatedAt"`
	BytesRecv   uint64    `json:"bytesRecv"`
	BytesSent   uint64    `json:"bytesSent"`
	Dropin      uint64    `json:"dropin"`
	Dropout     uint64    `json:"dropout"`
	Errin       uint64    `json:"errin"`
	Errout      uint64    `json:"errout"`
	Fifoin      uint64    `json:"fifoin"`
	Fifoout     uint64    `json:"fifoout"`
	Name        string    `json:"name"`
	PacketsRecv uint64    `json:"packetsRecv"`
	PacketsSent uint64    `json:"packetsSent"`
}

//func Updatediskio_db() {
//
//}

//func Creatediskio_db() int {
// Migrate the schema
//db.AutoMigrate(&Diskio{})
//
//}

//from https://dev.to/moficodes/build-your-first-rest-api-with-go-2gcj
//collectStatsCPU 1 second cpu time sample
func collectStatsCPU(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	//w.WriteHeader(http.StatusOK)
	percent, err := cpu.Percent(1*time.Second, false)
	if err != nil {
		fmt.Printf("error getting cpu stats: error: %v", err)
	}
	var u CPUStats
	u.Average = int(math.Round(percent[0]))
	json.NewEncoder(w).Encode(u)

}

//collectStatsMEM returns memory usage
func collectStatsMEM(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	v, err := mem.VirtualMemory()
	if err != nil {
		fmt.Printf("error getting mem stats: error: %v", err)
	}
	json.NewEncoder(w).Encode(v)

}

//collectStatsDISK used when the url is /disk/X: note, it will only take capitolized disk letters
func collectStatsDISK(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	pathParams := mux.Vars(r)
	driveltr := "null"
	val := pathParams["driveletter"]
	//fmt.Printf(" the drive letter passed to me was %v", val)

	checkedvar, err := regexp.MatchString("^[a-zA-Z]:$", val)
	if err != nil {
		fmt.Printf("incorrect args passed as drive letter: %v", err)
	}
	if checkedvar == true {
		driveltr = pathParams["driveletter"]
		v, err := disk.Usage(driveltr)
		if err != nil {
			fmt.Printf("error getting disk stats, error: %v", err)
		}
		json.NewEncoder(w).Encode(v)
		return
	}
}

//Iocounts will return input/output for all drives. counters are bases on since disk showed up after boot, or mounting.(they could zeroize)
func Iocounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	v, err := disk.IOCounters(":")
	if err != nil {
		//fmt.Printf("error is %v \n enabling disk stats interface with diskperf -y \n", err)
		var cmdargs1 = "diskperf -y"
		_, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
		if err != nil {
			fmt.Printf("failed to enable diskstats, error is : %v \n", err)
		}
	}
	json.NewEncoder(w).Encode(v)

}

//nCounters shows network traffic counters, since boot.
func nCounters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	v, err := net.IOCounters(true)
	if err != nil {
		fmt.Printf("error with network stats collection is %v \n ", err)

	}
	json.NewEncoder(w).Encode(v)

}

//pshell test function, can be removed
func pshell() []byte {
	//	var cmdargs1 = "Get-WmiObject win32_volume | Select-Object SystemName, BlockSize, Capacity, FreeSpace, DriveLetter , @{Name=\"CapacityGB\";Expression={[math]::round($_.Capacity/1GB,2)}}, @{Name=\"FreeSpaceGB\";Expression={[math]::round($_.FreeSpace/1GB,2)}} , @{Name=\"FreeSpacePercent\";Expression={[math]::round(($_.FreeSpace/($_.Capacity*1.00))*100.00,2)}} , @{Name=\"Date\";Expression={$(Get-Date -f s)}}| Sort-Object Name | Convertto-JSON"
	var cmdargs1 = "Get-WmiObject win32_volume | Select-Object Label, FileSystem, SystemVolume, BootVolume, SystemName, BlockSize, Capacity, FreeSpace, DriveLetter , @{Name=\"CapacityGB\";Expression={[math]::round($_.Capacity/1GB,2)}}, @{Name=\"FreeSpaceGB\";Expression={[math]::round($_.FreeSpace/1GB,2)}} , @{Name=\"FreeSpacePercent\";Expression={[math]::round(($_.FreeSpace/($_.Capacity*1.00))*100.00,2)}} , @{Name=\"Date\";Expression={$(Get-Date -f s)}}| Sort-Object Name |  where-object \"DriveLetter\" -notlike \"\" | where-object DriveLetter -notlike \"C:\"  | where-object BlockSize -notlike \"\" | where-object FileSystem -like \"NTFS\" | where-object Label -notlike \"System\" | Convertto-JSON"
	out, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		//		log.Print("error: %v \n" ,&err)
		log.Writer()
	}
	//if no length, an error happened or, there were no drives discovered. handle it with blank object handed back to api.
	sz := len(out)
	if 0 == sz {
		//a := []byte("{}")
		//fmt.Println("NO drives found")
		//return a
		errmsg := fmt.Sprintln(`{"message": "No Drives Found"}`)
		fmt.Printf("%v", errmsg)
		return []byte(errmsg)
	}
	//else return the json output from the powershell
	return out
}

//collectStatsHome prints a templated stats page in html, may not be used. just a test
func collectStatsHome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `<!doctype html>
<html lang="en">
  <head>
    <!-- Required meta tags -->
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">

    <!-- Bootstrap CSS -->
    <link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.5.0/css/bootstrap.min.css" integrity="sha384-9aIt2nRpC12Uk9gS9baDl411NQApFmC26EwAOH8WgZl5MYYxFfc+NcPb1dKGj7Sk" crossorigin="anonymous">

    <title>EDOS Status</title>
  </head>
  <body style="background-color: #212121;">
  <script src="https://code.highcharts.com/highcharts.js"></script>
  <div id="container" style="height: 300px"></div>
<div class="row fluid-img">
<br><br>
      
	</div>
	<!-- Optional JavaScript -->
	<script>document.addEventListener('DOMContentLoaded', function () {
        var myChart = Highcharts.chart('container', {
            chart: {
                type: 'bar'
            },
            title: {
                text: 'Fruit Consumption'
            },
            xAxis: {
                categories: ['Apples', 'Bananas', 'Oranges']
            },
            yAxis: {
                title: {
                    text: 'Fruit eaten'
                }
            },
            series: [{
                name: 'Jane',
                data: [1, 0, 4]
            }, {
                name: 'John',
                data: [5, 7, 3]
            }]
        });
    });</script>
    <!-- jQuery first, then Popper.js, then Bootstrap JS -->
    <script src="https://code.jquery.com/jquery-3.5.1.slim.min.js" integrity="sha384-DfXdz2htPH0lsSSs5nCTpuj/zy4C+OGpamoFVy38MVBnE+IbbVYUew+OrCXaRkfj" crossorigin="anonymous"></script>
    <script src="https://cdn.jsdelivr.net/npm/popper.js@1.16.0/dist/umd/popper.min.js" integrity="sha384-Q6E9RHvbIyZFJoft+2mJbHaEWldlvI9IOYy5n3zV9zzTtmI3UksdQRVvoxMfooAo" crossorigin="anonymous"></script>
	<script src="https://stackpath.bootstrapcdn.com/bootstrap/4.5.0/js/bootstrap.min.js" integrity="sha384-OgVRvuATP1z7JjHLkuOU7Xw704+h835Lr+6QL9UvYjZE3Ipu6Tp75j7Bh/kR0JKI" crossorigin="anonymous"></script>

  </body>
</html>
`)

}

//getexportsps is the powershell command to get the current nfs exports listing
func getexportsps() []byte {

	//	uses struct ExportsJson or ExportsJsonArray 2+ result is

	exportarrayresulttestdata := []byte(`
	[ {
           "name":  "powershell nfs missing",
           "anonymousaccess":  true,
           "anonymousuid":  65534,
           "anonymousgid":  65534,
           "isonline":  true,
           "path":  "D:\\"
       },
       {
           "name":  "Please install nfs",
           "anonymousaccess":  false,
           "anonymousuid":  -2,
           "anonymousgid":  -2,
           "isonline":  true,
           "path":  "H:\\"
       }
   ]`)

	//	1 export result is
	exportnoarraytestdata := []byte(`   		{
              "name":  "test",
              "anonymousaccess":  true,
              "anonymousuid":  65534,
              "anonymousgid":  65534,
              "isonline":  true,
              "path":  "D:\\"
          }`)

	//	no export result is literally null.
	exportnoexportstestdata := []byte(``)

	//here we run powershell to get the current nfs shares, this is mostly nice due to the ability to have it already json content.
	var cmdargs1 = "Get-NfsShare |select name, anonymousaccess, anonymousuid, anonymousgid, isonline, path | Convertto-JSON"
	//var cmdargs1 = "Get-WmiObject win32_volume| Convertto-JSON"
	out, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		//maybe powershell fails, or nfs modules not installed. use this for test case static definitions here.
		out1 := exportarrayresulttestdata
		out2 := exportnoarraytestdata
		out3 := exportnoexportstestdata
		//which test case
		out = out1
		_ = out2
		_ = out3
	}

	//check if the result contains an array bracket, if so we know its an array result, else not an array.
	if bytes.Contains(out, []byte("[")) {
		//		fmt.Printf(" json array detected, using array struct\n")
		var res []ExportsJSON
		//		fmt.Println("just a test +\n", "1")
		json.Unmarshal(out, &res)
		//		fmt.Printf(" no loop match %+v", res)
		for _, re := range res {
			fmt.Printf("multiple exports found, name is %s \n", re.Name)
		}
	} else {
		var res ExportsJSON
		json.Unmarshal(out, &res)
		fmt.Printf("single export found, name is  %s\n", res.Name)
	}

	//if no length, an error happened or, there were no exports discovered. handle it with blank object handed back to api.
	sz := len(out)
	if 0 == sz {
		a := []byte("{}")
		fmt.Println("NO exports found")
		return a
	}
	return out

}

func printSlice(s []byte) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}

func importdrivepacksps() []byte {
	var cmdargs1 = "/c1 /fall import j"
	out, err := exec.Command("C:\\Program Files (x86)\\MegaRAID Storage Manager\\storcli64.exe", cmdargs1).Output()
	if err != nil {
		//		log.Print("error: %v \n" ,&err)
		errfixed := strings.ReplaceAll(err.Error(), `"`, `\"`)
		log.Printf("error import foreign array. error: %v \n", errfixed)
		errmsg := fmt.Sprintf(`{"message": "Error importing foreign array. error: %v \n"}`, errfixed)
		fmt.Printf("%v", errmsg)
		return []byte(errmsg)
	}
	return out
}

//MakeDrivesUnconfiguredGoodps is the powershell script to tell the raid controller to change the state to good. will error if already good, so returns are trivial and need inspection
func MakeDrivesUnconfiguredGoodps() []byte {
	var cmdargs1 = "/c1 /e245 /sall set good j"
	out, err := exec.Command("C:\\Program Files (x86)\\MegaRAID Storage Manager\\storcli64.exe", cmdargs1).Output()
	if err != nil {
		//		log.Print("error: %v \n" ,&err)
		errfixed := strings.ReplaceAll(err.Error(), `"`, `\"`)
		fmt.Printf("error making drives unconfigured good. error: %v \n", errfixed)
		errmsg := fmt.Sprintf(`{"message": "error making drives unconfigured good. error: %v \n"}`, errfixed)
		return []byte(errmsg)
	}
	return out
}

//getraiddrivestatusps is the powershell script to get the status of all drives on the raid card.
func getraiddrivestatusps() []byte {
	//just get the output of the drives, dont go into a logic rabbit hole to try to fix, let the user take actions off the data.
	var cmdargs1 = "/c1 show j"
	out, err := exec.Command("C:\\Program Files (x86)\\MegaRAID Storage Manager\\storcli64.exe", cmdargs1).Output()
	if err != nil {
		errfixed := strings.ReplaceAll(err.Error(), `"`, `\"`)
		fmt.Printf("Error showing drive status, error: %v \n", errfixed)
		errmsg := fmt.Sprintf(`{"message": "Error showing drive status, error: %v \n"}`, errfixed)
		return []byte(errmsg)
	}
	return out
}

//GetUnitInfops is the powershell script to get all unit status (degraded, optimal etc)
func GetUnitInfops() []byte {
	//gets unit info, only shows if the unit was imported
	var cmdargs1 = "/c1 /vall show j"
	out, err := exec.Command("C:\\Program Files (x86)\\MegaRAID Storage Manager\\storcli64.exe", cmdargs1).Output()
	if err != nil {
		errfixed := strings.ReplaceAll(err.Error(), `"`, `\"`)
		fmt.Printf(`{"message": "error showing unit info, error: %v \n"}`, errfixed)
		errmsg := fmt.Sprintf(`{"message": "error showing unit info, error: %v \n"}`, errfixed)
		return []byte(errmsg)
	}
	log.Printf("GetUnitInfo returned: \n %s", out)
	return out
}

//
func createraidAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	pathParams := mux.Vars(r)
	ports := pathParams["ports"]
	name := pathParams["name"]

	fmt.Printf("ports is %v\n name is %v\n", ports, name)
	m, err := regexp.MatchString("[0]\\-[3]|[4]\\-[7]|[8]\\-[1][1]", ports)
	fmt.Printf("m is %v", m)
	if err != nil || m != true {
		errmsg := `{"message": "Drive numbers not recognized.only 0 - 3,4 - 7,8 - 11 are accepted."}`
		fmt.Printf("%v", errmsg)
		w.Write([]byte(errmsg))
		return
	}

	var cmdargs1 = fmt.Sprintf("/c1 add vd type=r5 name=%v drives=245:%v SED direct j", name, ports)
	out, err := exec.Command("C:\\Program Files (x86)\\MegaRAID Storage Manager\\storcli64.exe", cmdargs1).Output()
	if err != nil {
		log.Printf("error creating raid, error: %v \n", err)
		//errstring := err.Error()
		errfixed := strings.ReplaceAll(err.Error(), `"`, `\"`)
		errmsg := string(fmt.Sprintf(`{"message": "Error creating raid, error: %v"}`, errfixed))
		fmt.Printf("%v", errmsg)
		w.Write([]byte(errmsg))
		return
	}
	fmt.Printf(`{"message": "Createraid command output: %v"}`, string(out))
	//json.NewEncoder(w).Encode(out) //not needed, output is already json
	w.Write(out)
	return
}

//rescandiskps is the powershell to rescan all disks on the raid controller.
func rescandisksps() []byte {
	//rescans the disks
	var cmdargs1 = "/c1 restart j"
	out, err := exec.Command("C:\\Program Files (x86)\\MegaRAID Storage Manager\\storcli64.exe", cmdargs1).Output()
	if err != nil {
		//		log.Print("error: %v \n" ,&err)
		//log.Writer()
		//log.Printf("error rescanning disks on /c1, error: %v \n", err)
		errfixed := strings.ReplaceAll(err.Error(), `"`, `\"`)
		errmsg := string(fmt.Sprintf(`{"message": "error rescanning disks on /c1, error: %v \n"}`, errfixed))
		fmt.Printf("%v", errmsg)
		return []byte(errmsg)
	}
	log.Printf("controller restart returned: \n %s", out)
	return out
}

func get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "get called"}`))
}

func status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(pshell())
}

func getexports(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(getexportsps())
}

func importdrivepacks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(importdrivepacksps())
}

func getraiddrivestatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	w.Write(getraiddrivestatusps())
}

//MakeDrivesUnconfiguredGood will try to change status to good on all drives attached. (in hopes of changing unconfigured bad to unconfigured good)
func MakeDrivesUnconfiguredGood(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	w.Write(MakeDrivesUnconfiguredGoodps())
}

//GetUnitInfo will get the status of the current raids
func GetUnitInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	w.Write(GetUnitInfops())
}

func rescandisks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	w.Write(rescandisksps())
}

//func params(w http.ResponseWriter, r *http.Request) {
//	pathParams := mux.Vars(r)
//	w.Header().Set("Content-Type", "application/json")
//
//	userID := -1
//	var err error
//	if val, ok := pathParams["userID"]; ok {
//		userID, err = strconv.Atoi(val)
//		if err != nil {
//			w.WriteHeader(http.StatusInternalServerError)
//			w.Write([]byte(`{"message": "need a number"}`))
//			return
//		}
//	}
//	commentID := -1
//	if val, ok := pathParams["commentID"]; ok {
//		commentID, err = strconv.Atoi(val)
//		if err != nil {
//			w.WriteHeader(http.StatusInternalServerError)
//			w.Write([]byte(`{"message": "need a number"}`))
//			return
//		}
//	}
//	query := r.URL.Query()
//	location := query.Get("location")
//	w.Write([]byte(fmt.Sprintf(`{"userID": %d, "commentID": %d, "location": "%s" }`, userID, commentID, location)))
//}

//CollectDiskIoStatsInBackground will gather disk io stats per 10 seconds (or value called)
//store in the db, for around 500 iterations
func CollectDiskIoStatsInBackground(driveltr string) {
	db, err := gorm.Open("sqlite3", `C:\ProgramData\edosAPI\edosapi.db`)
	if err != nil {
		//	panic("failed to connect database")
		fmt.Printf("failed to attach to database %v \n", `C:\ProgramData\edosAPI\edosapi.db`)
	}
	defer db.Close()

	diskiodata, err := disk.IOCounters(driveltr)
	if err != nil {
		//fmt.Printf("error is %v \n enabling disk stats interface with diskperf -y \n", err)
		var cmdargs1 = "diskperf -y"
		_, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
		if err != nil {
			fmt.Printf("failed to enable diskstats, error is : %v \n", err)
		}
	}
	if diskiodata[driveltr].Name != "" {
		//fmt.Printf("\nprinting io for drive %v: %v\n", driveltr, diskiodata[driveltr])
		//fmt.Printf("readCount = \nPath: %v, \nmergedReadCount: %v, \nwriteCount: %v, \nmergedwriteCount: %v, \nreadBytes: %v, \nreadTime: %v, \nwriteTime: %v, \niopsInProgress: %v, \nioTime: %v, \nweightedIO: %v, \nname: %v, \nserialNumber: %v, \nlabel: %v\n", diskiodata.readCount, diskiodata.mergedReadCount, diskiodata.writeCount, diskiodata.mergedWriteCount, diskiodata.readBytes, diskiodata.writeBytes, diskiodata.readTime, diskiodata.writeTime, diskiodata.iopsInProgress, diskiodata.IoTime, diskiodata.weightedIO, diskiodata.name, diskiodata.serialNumber, diskiodata.label)
		db.AutoMigrate(&Diskio{})
		db.Create(&Diskio{ReadCount: diskiodata[driveltr].ReadCount,
			MergedReadCount:  diskiodata[driveltr].WriteCount,
			WriteCount:       diskiodata[driveltr].WriteCount,
			MergedWriteCount: diskiodata[driveltr].MergedWriteCount,
			ReadBytes:        diskiodata[driveltr].ReadBytes,
			WriteBytes:       diskiodata[driveltr].WriteBytes,
			ReadTime:         diskiodata[driveltr].ReadTime,
			WriteTime:        diskiodata[driveltr].WriteTime,
			IopsInProgress:   diskiodata[driveltr].IopsInProgress,
			IoTime:           diskiodata[driveltr].IoTime,
			WeightedIO:       diskiodata[driveltr].WeightedIO,
			Name:             diskiodata[driveltr].Name,
			SerialNumber:     diskiodata[driveltr].SerialNumber,
			Label:            diskiodata[driveltr].Label})
	}
}

//CollectNetworkIoStatsInBackground will store network stats in the sqllite db, every 5-10 seconds for 2 hours
func CollectNetworkIoStatsInBackground() {
	db, err := gorm.Open("sqlite3", `C:\ProgramData\edosAPI\edosapi.db`)
	if err != nil {
		//	panic("failed to connect database")
		fmt.Printf("failed to attach to database %v \n", `C:\ProgramData\edosAPI\edosapi.db`)
	}
	defer db.Close()
	db.AutoMigrate(&NetworkIoStats{})
	nstats, err := net.IOCounters(true)
	if err != nil {
		fmt.Printf("error getting network stats, error: %v", err)
	}
	for _, v := range nstats {
		if v.BytesRecv != 0 {
			//fmt.Printf("name: %v\n", v.Name)
			db.Create(&NetworkIoStats{
				BytesRecv:   v.BytesRecv,
				BytesSent:   v.BytesSent,
				Dropin:      v.Dropin,
				Dropout:     v.Dropout,
				Errin:       v.Errin,
				Errout:      v.Errout,
				Fifoin:      v.Fifoin,
				Fifoout:     v.Fifoout,
				Name:        v.Name,
				PacketsRecv: v.PacketsRecv,
				PacketsSent: v.PacketsSent})
		}
		//fmt.Printf("zero value on name: %v\n", v.Name)
	}
}

//CollectcpustatsInBackground will store cpu stats in the sqllite db, every 5-10 seconds for 2 hours
func CollectcpustatsInBackground() {
	db, err := gorm.Open("sqlite3", `C:\ProgramData\edosAPI\edosapi.db`)
	if err != nil {
		//	panic("failed to connect database")
		fmt.Printf("failed to attach to database %v \n", `C:\ProgramData\edosAPI\edosapi.db`)
	}
	defer db.Close()
	db.AutoMigrate(&CPUStats{})
	percent, err := cpu.Percent(1*time.Second, false)
	if err != nil {
		fmt.Printf("error getting cpu stats, error: %v", err)
	}
	var u CPUStats
	u.Average = int(math.Round(percent[0]))
	fmt.Printf("cpu is %d\n", u.Average)
	db.Create(&CPUStats{
		Average: u.Average})

	//for _, v := range percent {

	//	db.Create(&CPUStats1{
	//		Average: v.	})
	//	fmt.Printf("value on name: %v\n", v.Average)
	//}
}

//CollectmemstatsInBackground will store cpu stats in the sqllite db, every 5-10 seconds for 2 hours
func CollectmemstatsInBackground() {
	db, err := gorm.Open("sqlite3", `C:\ProgramData\edosAPI\edosapi.db`)
	if err != nil {
		//	panic("failed to connect database")
		fmt.Printf("failed to attach to database %v \n", `C:\ProgramData\edosAPI\edosapi.db`)
	}
	defer db.Close()
	db.AutoMigrate(&MemStats1{})
	//percent, err := cpu.Percent(1*time.Second, false)
	if err != nil {
		fmt.Printf("error getting Memory stats, error: %v", err)
	}
	//var u MemStats1
	//u.Average = int(math.Round(percent[0]))
	u, err := mem.VirtualMemory()
	//u.UsedPercent = int(math.Round(u.UsedPercent))
	fmt.Printf("Mem is %d\n", int(math.Round(u.UsedPercent)))
	db.Create(&MemStats1{
		Available:   u.Available,
		Used:        u.Used,
		UsedPercent: u.UsedPercent,
		Free:        u.Free,
		Active:      u.Active,
		Inactive:    u.Inactive})

	//for _, v := range percent {
	//	db.Create(&CPUStats1{
	//		Average: v.	})
	//	fmt.Printf("value on name: %v\n", v.Average)
	//}
}

//getnetworkstatsfromdb will present json data to make a graph with
func getnetworkstatsfromdb(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	db, err := gorm.Open("sqlite3", `C:\ProgramData\edosAPI\edosapi.db`)
	if err != nil {
		//	panic("failed to connect database")
		fmt.Printf("failed to attach to database %v \n", `C:\ProgramData\edosAPI\edosapi.db`)
	}
	defer db.Close()
	type Results struct {
		Total string
	}
	var sqldata []NetworkIoStats

	db.Table("network_io_stats").Select("*").Scan(&sqldata)
	//db.Table("network_io_stats").Find(&sqldata)
	emp := &sqldata
	e, err := json.Marshal(emp)
	w.Write([]byte(e))
}

//getdiskstatsfromdb will present json data to make a graph with
func getdiskstatsfromdb(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	db, err := gorm.Open("sqlite3", `C:\ProgramData\edosAPI\edosapi.db`)
	if err != nil {
		//	panic("failed to connect database")
		fmt.Printf("failed to attach to database %v \n", `C:\ProgramData\edosAPI\edosapi.db`)
	}
	defer db.Close()
	type Results struct {
		Total string
	}
	var sqldata []DiskStats

	db.Table("disk_stats").Select("*").Scan(&sqldata)
	//db.Table("network_io_stats").Find(&sqldata)
	emp := &sqldata
	e, err := json.Marshal(emp)
	w.Write([]byte(e))
}

//getmemstatsfromdb will present json data to make a graph with
func getmemstatsfromdb(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	db, err := gorm.Open("sqlite3", `C:\ProgramData\edosAPI\edosapi.db`)
	if err != nil {
		//	panic("failed to connect database")
		fmt.Printf("failed to attach to database %v \n", `C:\ProgramData\edosAPI\edosapi.db`)
	}
	defer db.Close()
	var sqldata []MemStats1
	db.Table("mem_stats1").Select("*").Scan(&sqldata)
	//db.Table("mem_stats").Find(&sqldata)
	emp := &sqldata
	e, err := json.Marshal(emp)
	w.Write([]byte(e))
}

//getcpustatsfromdb will present json data to make a graph with
func getcpustatsfromdb(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	db, err := gorm.Open("sqlite3", `C:\ProgramData\edosAPI\edosapi.db`)
	if err != nil {
		//	panic("failed to connect database")
		fmt.Printf("failed to attach to database %v \n", `C:\ProgramData\edosAPI\edosapi.db`)
	}
	defer db.Close()

	var sqldata []CPUStats

	db.Table("cpu_stats").Select("*").Scan(&sqldata)
	//db.Table("network_io_stats").Find(&sqldata)
	emp := &sqldata
	e, err := json.Marshal(emp)
	w.Write([]byte(e))
}

//getdiskiostatsfromdb will present json data to make a graph with
func getdiskiotatsfromdb(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	db, err := gorm.Open("sqlite3", `C:\ProgramData\edosAPI\edosapi.db`)
	if err != nil {
		//	panic("failed to connect database")
		fmt.Printf("failed to attach to database %v \n", `C:\ProgramData\edosAPI\edosapi.db`)
	}
	defer db.Close()
	type Results struct {
		Total string
	}
	var sqldata []Diskio

	db.Table("network_io_stats").Select("*").Scan(&sqldata)
	//db.Table("network_io_stats").Find(&sqldata)
	emp := &sqldata
	e, err := json.Marshal(emp)
	w.Write([]byte(e))
}

//CollectDiskstatsInBackground will collect the driveltr passed disk (partition) stats, and store in sqllite db under drive_stats table.
//intended to be called by a threaded function for each disk.
func CollectDiskstatsInBackground(driveltr string) uint {
	//fmt.Print(driveltr)
	//fmt.Print(keepduration)
	db, err := gorm.Open("sqlite3", `C:\ProgramData\edosAPI\edosapi.db`)
	if err != nil {
		//	panic("failed to connect database")
		fmt.Printf("failed to attach to database %v \n", `C:\ProgramData\edosAPI\edosapi.db`)
	}
	defer db.Close()

	diskdata, err := disk.Usage(driveltr)
	if err != nil {
		fmt.Printf("error getting disk stats, error: %v", err)
	}

	db.AutoMigrate(&DiskStats{})

	// Create
	//fmt.Printf("Disk Stats = \nPath: %v, \nFstype: %v, \nTotal: %v, \nFree: %v, \nUsed: %v, \nUsedPercent: %v, \nInodesTotal: %v, \nInodesUsed: %v, \nInodesFree: %v, \nInodesUsedPercent: %v", diskdata.Path, diskdata.Fstype, diskdata.Total, diskdata.Free, diskdata.Used, diskdata.UsedPercent, diskdata.InodesTotal, diskdata.InodesUsed, diskdata.InodesFree, diskdata.InodesUsedPercent)

	db.Create(&DiskStats{Path: diskdata.Path,
		Fstype: diskdata.Fstype,
		Total:  diskdata.Total, Free: diskdata.Free,
		Used:              diskdata.Used,
		UsedPercent:       diskdata.UsedPercent,
		InodesTotal:       diskdata.InodesTotal,
		InodesUsed:        diskdata.InodesUsed,
		InodesFree:        diskdata.InodesFree,
		InodesUsedPercent: diskdata.InodesUsedPercent})

	//fmt.Printf("Path: %v, Fstype: %v, Total: %v, Free: %v, Used: %v, UsedPercent: %v, InodesTotal: %v, InodesUsed: %v, InodesFree: %v, InodesUsedPercent: %v", diskstats.Path, diskdata.Fstype, diskdata.Total, diskdata.Free, diskdata.Used, diskdata.UsedPercent, diskdata.InodesTotal, diskdata.InodesUsed, diskdata.InodesFree, diskdata.InodesUsedPercent)

	var firstrecord DiskStats
	var lastrecord DiskStats
	type Results struct {
		Total uint
	}
	var rowtotal Results
	//db.LogMode(true)
	db.First(&firstrecord, "Path = ?", driveltr).Where("Path = ?", driveltr)
	db.Last(&lastrecord, "Path = ?", driveltr).Where("Path = ?", driveltr)
	//get total rows for this disk in the db
	db.Table("disk_stats").Select("count(id) as total").Where("Path = ?", driveltr).Scan(&rowtotal)
	fmt.Printf("\ntotal row count for drive %v is %v\n", driveltr, rowtotal.Total)

	//now take that, and delete all records with that path, older then the keepduration. limiting the storage used.
	if rowtotal.Total > 100 {

		dl := db.Unscoped().Delete(&DiskStats{}, "Path = ? AND created_at < datetime('now', '-30 days')", driveltr)

		fmt.Printf("deleted old disk_stats rows = %v\n", dl.RowsAffected)
	}
	return firstrecord.ID
}

//PurgeDbRecordsDiskios will delete data from the sqllite db
func PurgeDbRecordsDiskios(duration string) {
	//fmt.Printf("table: %v, duration: %v, direction: %v\n", table, duration, direction)
	db, err := gorm.Open("sqlite3", `C:\ProgramData\edosAPI\edosapi.db`)
	if err != nil {
		//	panic("failed to connect database")
		fmt.Printf("failed to attach to database %v \n", `C:\ProgramData\edosAPI\edosapi.db`)
	}
	defer db.Close()
	//just use raw sql, the library hates me
	//db.LogMode(true)
	result := db.Exec("DELETE from diskios where created_at < datetime('now', ?);", duration)
	fmt.Printf("deleted old diskios rows = %v\n", result.RowsAffected)
}

//PurgeDbRecordsCPU will delete data from the sqllite db
func PurgeDbRecordsCPU(duration string) {
	//fmt.Printf("table: %v, duration: %v, direction: %v\n", table, duration, direction)
	db, err := gorm.Open("sqlite3", `C:\ProgramData\edosAPI\edosapi.db`)
	if err != nil {
		//	panic("failed to connect database")
		fmt.Printf("failed to attach to database %v \n", `C:\ProgramData\edosAPI\edosapi.db`)
	}
	defer db.Close()
	//just use raw sql, the library hates me
	//db.LogMode(true)
	result := db.Exec("DELETE from cpu_stats where created_at < datetime('now', ?);", duration)
	fmt.Printf("deleted old CPU stats rows = %v\n", result.RowsAffected)
}

//PurgeDbRecordsMEM will delete data from the sqllite db
func PurgeDbRecordsMEM(duration string) {
	db, err := gorm.Open("sqlite3", `C:\ProgramData\edosAPI\edosapi.db`)
	if err != nil {
		//	panic("failed to connect database")
		fmt.Printf("failed to attach to database %v \n", `C:\ProgramData\edosAPI\edosapi.db`)
	}
	defer db.Close()
	result := db.Exec("DELETE from mem_stats1 where created_at < datetime('now', ?);", duration)
	fmt.Printf("deleted old mem_stats rows = %v\n", result.RowsAffected)
}

//PurgeDbRecordsNetworkIoStats will delete data from the sqllite db
func PurgeDbRecordsNetworkIoStats(duration string) {
	//fmt.Printf("table: %v, duration: %v, direction: %v\n", table, duration, direction)
	db, err := gorm.Open("sqlite3", `C:\ProgramData\edosAPI\edosapi.db`)
	if err != nil {
		//	panic("failed to connect database")
		fmt.Printf("failed to attach to database %v \n", `C:\ProgramData\edosAPI\edosapi.db`)
	}
	defer db.Close()
	//just use raw sql, the library hates me
	//db.LogMode(true)
	result := db.Exec("DELETE from network_io_stats where created_at < datetime('now', ?);", duration)
	fmt.Printf("deleted old network stats rows = %v\n", result.RowsAffected)
}

func getalldrives() []string {
	var drives []string
	partitions, _ := disk.Partitions(false)
	for _, partition := range partitions {
		//	fmt.Println(partition.Mountpoint)
		drives = append(drives, partition.Mountpoint)
	}
	return drives
}

//initializedisk will try to put a partition table on a disk, returning exit code on, 0 for success, 1 for fail
func initializedisk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	pathParams := mux.Vars(r)
	drivenumber := pathParams["drivenumber"]

	fmt.Printf("drivenumber is %v\n", drivenumber)
	m, err := regexp.MatchString("[0-9]", drivenumber)
	//fmt.Printf("m is %v", m)
	if err != nil || m != true {
		errmsg := `{"message": "Drive number not recognized or not input. exiting."}`
		fmt.Printf("%v", errmsg)
		w.Write([]byte(errmsg))
		return
	}

	var cmdargs1 = fmt.Sprintf("Initialize-Disk -Number %v |Convertto-json", drivenumber)
	out, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		//		log.Print("error: %v \n" ,&err)
		if exitError, ok := err.(*exec.ExitError); ok {
			out := exitError.ExitCode()
			bs := []byte(strconv.Itoa(out))
			out2 := fmt.Sprintf("{\"status\": \"%s\"}", bs)
			bs2 := []byte(out2)

			//fmt.Println(bs)
			w.Write(bs2)
			//return exitError.ExitCode()
			log.Writer()
			return

		}
	} //
	out = []byte(`{"status": "0"}`)
	w.Write(out)
}

//
//
//function is incomplete, total pain to get the array info currently and dont want to waste time on it, when other functions are still needing to be worked.
//
//
func deleteraid(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	pathParams := mux.Vars(r)
	ports := pathParams["ports"]

	fmt.Printf("raided ports to delete are %v\n", ports)
	m, err := regexp.MatchString("[0]\\-[3]|[4]\\-[7]|[8]\\-[1][1]", ports)
	//fmt.Printf("m is %v", m)
	if err != nil || m != true {
		errmsg := `{"message": "Drive numbers not recognized.only 0 - 3,4 - 7,8 - 11 are accepted."}`
		fmt.Printf("%v", errmsg)
		w.Write([]byte(errmsg))
		return
	}
	msg := fmt.Sprintf(`{"message": "finding which drive group uses ports %v\n"}`, ports)
	fmt.Printf("%v", msg)
	//w.Write([]byte(msg))

	//get the drive group of each drive asked for, make sure they are the same group or error
	//get first number
	//fmt.Println(string(ports[0]))
	i, err := strconv.Atoi(string(ports[0]))
	if err != nil {
		fmt.Printf("conversion error = %v\n i is %d\n", err, i)
	}
	//now interate on it.. not needed for the grouping, but keeping for later use just in case
	//endport := i + 4
	//for i < endport {
	//	out := getdrivegroupfromportnumber(i)
	//	w.Write(out)
	//	fmt.Printf("port is %d\n", i)
	//	fmt.Printf("%v\n", out)
	//	i++
	//}
	out, err := getdrivegroupfromportnumber(ports)
	if err != nil {
		fmt.Printf("error finding drive group for ports %v, aborting", ports)
		errmsg := fmt.Sprintf(`{"message": "error finding drive group for ports %v,  exiting."}`, ports)
		fmt.Printf("%v", errmsg)
		w.Write([]byte(errmsg))
		return
	}
	fmt.Printf("%v\n", string(out))
	var res Drivegroupnumber
	json.Unmarshal([]byte(out), &res)
	//fmt.Printf("%v", res)
	//size := len(res)
	var intArray [4]int
	iteration := 0
	for _, res := range res {
		drivenumber := res.EIDSlot
		splitvar := strings.Split(drivenumber, ":")
		fmt.Printf("drive portnumber:%v=DG:%d \n", splitvar[1], res.DG)
		//add DG value to intArray for comparison after
		intArray[iteration] = res.DG
		iteration++
	}
	//test for equality of all elements in the array
	//fmt.Println(intArray[0])
	if (intArray[0] == intArray[1]) && (intArray[0] == intArray[2]) && (intArray[0] == intArray[3]) {
		fmt.Printf("all drivegroups match, ok to delete drive group %d\n ", intArray[0])
		fmt.Printf("locating virtual disk tied to drive group %v", intArray[0])
		virtdisk := getvirtdiskfromdrivegroup(intArray[0])
		del := removevirtualdisk(virtdisk)
		w.Write(del)
		return
	}
	w.Write(out)

}

//getvirtdiskfromdrivegroup will retrun the number of the virtual disk bound to drive group dg
func getvirtdiskfromdrivegroup(dg int) int {
	var cmdargs1 = fmt.Sprintf(`cd "C:\Program Files (x86)\MegaRAID Storage Manager\" ; $vd=.\storcli.exe /c1/vall show j | ConvertFrom-Json; $vd.Controllers."Response Data"."Virtual Drives"| where-object "DG/VD" -like "%v/*" |select-object "DG/VD"|ConvertTo-Json`, dg)
	out, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		msg := fmt.Sprintf(`{"message": "ERROR = %v"}`, err)
		fmt.Printf("%v", msg)
		//return []byte(msg)
		return 404
	} //
	var res Vdisk
	json.Unmarshal(out, &res)
	splitvar := strings.Split(res.DGVD, "/")
	fmt.Printf("virtual disk for drive group %v = %v\n", dg, splitvar[1])
	intNumber, _ := strconv.Atoi(splitvar[1])
	return intNumber
}

//Drivegroupnumber stores the results of the function getdrivegroupfromportnumber and or getindividualdrivegroupfromportnumber
type Drivegroupnumber []struct {
	DG      int    `json:"DG"`
	State   string `json:"State"`
	EIDSlot string `json:"EID:Slot"`
}

//Vdisk will map to the output of /c1/vall show
type Vdisk struct {
	DGVD string `json:"DG/VD"`
}

func removedrivegroupnumber(drivegroup int) []byte {
	var cmdargs1 = fmt.Sprintf("/c1/v%v del force j", drivegroup)
	fmt.Println(cmdargs1)
	out, err := exec.Command("C:\\Program Files (x86)\\MegaRAID Storage Manager\\storcli64.exe", cmdargs1).Output()
	if err != nil {
		errfixed := strings.ReplaceAll(err.Error(), `"`, `\"`)
		fmt.Printf(`{"message": "error deleting drivegroup, error: %v \n"}`, errfixed)
		errmsg := fmt.Sprintf(`{"message": "error deleting drivegroup, error: %v \n"}`, errfixed)
		return []byte(errmsg)
	}
	fmt.Printf("delete drivegroup returned: \n %s \n", out)
	return out
}

func removevirtualdisk(virtualdisk int) []byte {
	var cmdargs1 = fmt.Sprintf("/c1/v%v del force j", virtualdisk)
	fmt.Println(cmdargs1)
	out, err := exec.Command("C:\\Program Files (x86)\\MegaRAID Storage Manager\\storcli64.exe", cmdargs1).Output()
	if err != nil {
		errfixed := strings.ReplaceAll(err.Error(), `"`, `\"`)
		fmt.Printf(`{"message": "error deleting virtualdisk, error: %v \n"}`, errfixed)
		errmsg := fmt.Sprintf(`{"message": "error deleting virtualdisk, error: %v \n"}`, errfixed)
		return []byte(errmsg)
	}
	fmt.Printf("delete virtualdisk returned: \n %s \n", out)
	return out
}

func getdrivegroupfromportnumber(portrange string) ([]byte, error) {
	//damn ms like sucks for regex, make different command for the port range of 8-11
	var cmdargs1 = fmt.Sprintf(`cd "C:\Program Files (x86)\MegaRAID Storage Manager\" ; $dg=.\storcli.exe /c1 show j | ConvertFrom-Json; $dg.Controllers."Response Data"."TOPOLOGY"| where-object EID:SLOT -like 245:[%v] |select-object DG, State, EID:Slot|ConvertTo-Json`, portrange)

	if portrange == "8-11" {
		cmdargs1 = fmt.Sprint(`cd "C:\Program Files (x86)\MegaRAID Storage Manager\" ; $dg=.\storcli.exe /c1 show j | ConvertFrom-Json; $dg.Controllers."Response Data"."TOPOLOGY"| where-object EID:SLOT -like 245:* |where-object EID:SLOT -notlike 245:[0-7] |select-object DG, State, EID:Slot|ConvertTo-Json`)
	}
	out, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		msg := fmt.Sprintf(`{"message": "ERROR = %v"}`, err)
		fmt.Printf("%v", msg)

		return []byte(msg), errors.New(msg)
	} //

	//was the data from the card valid? no length = no
	sz := len(out)
	if sz == 0 {
		fmt.Printf("ERROR = out length is %v\n drive group not found\n", sz)
		msg := fmt.Sprintf(`{"message": "ERROR = out length is %v\n drive group not found\n"}`, sz)
		return []byte(msg), errors.New(msg)
	}

	//time.Sleep(20 * time.Second)
	//os.Exit(1)
	return out, nil
}

func getindividualdrivegroupfromportnumber(portnumber int) []byte {
	var cmdargs1 = fmt.Sprintf(`cd "C:\Program Files (x86)\MegaRAID Storage Manager\" ; $dg=.\storcli.exe /c1 show j | ConvertFrom-Json; $dg.Controllers."Response Data"."TOPOLOGY"| where-object EID:SLOT -eq 245:%v |select-object DG, State, EID:Slot|ConvertTo-Json`, portnumber)
	out, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		msg := fmt.Sprintf(`{"message": "ERROR = %v"}`, err)
		fmt.Printf("%v", msg)
		return []byte(msg)
	} //
	return out
}

func initializediskps(drivenumber string) int {

	var cmdargs1 = fmt.Sprintf("Initialize-Disk -Number %v |Convertto-json", drivenumber)
	_, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		log.Writer()
		return 0
	} //

	//now check to see if it worked or not, we have a function for that already
	diditwork := Checkdiskinitialized(drivenumber)
	return diditwork
}

//Checkdiskinitialized checks to see if the disk is initialized or not, used by initandpartitionandformatandnamedisk
func Checkdiskinitialized(drivenumber string) int {

	var cmdargs1 = fmt.Sprintf("get-disk | where-object number -eq %v |where-object PartitionStyle -eq 'RAW'|measure-object| %% { $_.Count }", drivenumber)
	out, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		fmt.Printf("error: %v \n", &err)
		return 0
	}
	fmt.Printf("output of checkdisksinitialized %s\n", out)
	//byteNumber := []byte("0")
	//fmt.Print(out)
	//fmt.Println(out)
	intNumber, _ := strconv.Atoi(string(out[0]))
	fmt.Printf("raw disks found = %d\n", intNumber)
	return int(intNumber)
}

//Checkdiskinitialized checks to see if the disk is initialized or not, used by initandpartitionandformatandnamedisk
func wipedisk(drivenumber string) int {
	outint := 0
	var cmdargs1 = fmt.Sprintf("clear-disk -Number %v -removedata -confirm:$false", drivenumber)
	_, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		fmt.Printf("wipe disk error: %v \n", &err)
		return 0
	}
	//if bad exit code return that, else return 0
	if exitError, ok := err.(*exec.ExitError); ok {
		outint = exitError.ExitCode()
		fmt.Printf("Wipedisk error happened, exit code %d\n", outint)
		return outint
	}
	fmt.Printf("Wipedisk function return is %d\n", outint)
	return outint
}

//checkforpartition checks and counts partitions on a disk number
func checkforpartition(drivenumber string) int {
	var cmdargs1 = fmt.Sprintf("get-partition | where-object DiskNumber -eq %v |measure-object| %% { $_.Count }", drivenumber)
	out, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		fmt.Printf("partition check error: %v \n", &err)
		return 0
	}
	//byteNumber := []byte("0")
	intNumber, _ := strconv.Atoi(string(out[0]))
	fmt.Printf("check if partitions exist = %d\n", intNumber)
	return int(intNumber)
}

func doesdriveexists(drivenumber string) int {
	var cmdargs1 = fmt.Sprintf("get-disk | where-object Number -eq %v | where-object IsSystem -eq $false |measure-object| %% { $_.Count }", drivenumber)
	out, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		fmt.Printf("doesdriveexists error: %v \n", &err)
		return 0
	}
	//byteNumber := []byte("0")
	intNumber, _ := strconv.Atoi(string(out[0]))
	fmt.Printf("check if drive exist = %d\n", intNumber)
	return int(intNumber)
}

func getpsdriveinfo(w http.ResponseWriter, r *http.Request) {
	var cmdargs1 = "get-disk | where-object IsSystem -eq $false|Convertto-json"
	out, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		//		log.Print("error: %v \n" ,&err)
		log.Writer()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(out)
}

func listpaths(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"urls": 
[{
"url1": "/mem", 
"url2": "/getmemstats",
"url3": "/cpu", 
"url4": "/getcpustats",
"url5": "/disk/{driveletter:}", 
"url6": "/diskio", 
"url7": "/getdiskio",
"url8": "/getdiskiostats",
"url9": "/networkio",
"url10": "/getnetworkio",
"url11": "/getexports", 
"url12": "/getraiddrivestatus", 
"url13": "/getunitinfo",
"url14": "/getpsdriveinfo",
"url15": "/status"
}] }`))
}

func listpathsauthed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"urls": 
[{
"url1": "/addexports/{driveletter:}/{drivename}",
"url2": "/removeexports/{exportname}",
"url3": "/importdrivepacks", 
"url4": "/makeunconfiguredgood",
"url5": "/rescandisks",
"url6": "/createraid/0-3 ( or 4-7 or 8-11 )/{raidname} like vnir or mwir",
"url7": "/deleteraid/0-3 ( or 4-7 or 8-11 )",
"url8": "/initializedisk/{drive number}",
"url9": "/initandpartitionandformatandnamedisk/{drive number}/{drive name}"
}] }`))
}

func exitnow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	os.Exit(1)
}

//partitionandformatandnamedisknew  will try to put a partition table on a disk, format it, and label it, and return the json output of the success or json message on fail.
func initandpartitionandformatandnamedisknew(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	pathParams := mux.Vars(r)
	drivenumber := pathParams["drivenumber"]
	drivename := pathParams["drivename"]

	//check drive number is sent
	fmt.Printf("drivenumber is %v\n", drivenumber)
	number, err := regexp.MatchString("[0-9]", drivenumber)
	//fmt.Printf("m is %v", m)
	if err != nil || number != true {
		errmsg := `{"message": "Drive number not recognized or not input. exiting."}`
		fmt.Printf("%v", errmsg)
		w.Write([]byte(errmsg))
		return
	}

	//check drive number exists on the system or abort 1=good, 0=bad, missing or system disk
	doesdriveexists := doesdriveexists(drivenumber)
	if doesdriveexists == 0 {
		//fmt.Printf("Drive number %v is not found or a system disk, aborting", drivenumber)
		errmsg := fmt.Sprintf(`{"message": "Drive number %v is not found or a system disk, aborting"}`, drivenumber)
		fmt.Printf("%v", errmsg)
		w.Write([]byte(errmsg))
		return
	}

	//check drivename is sent
	fmt.Printf("drivename is %v\n", drivename)
	name, err := regexp.MatchString("^[a-zA-Z]+$", drivename)
	//fmt.Printf("m is %v", m)
	if err != nil || name != true {
		errmsg := `{"message": "Drive name not recognized or has bad characters, A-Z and a-z only, aka vnir or VNIR not /vnir. exiting."}`
		fmt.Printf("%v", errmsg)
		w.Write([]byte(errmsg))
		return
	}

	//wipe it and reinit to destroy previous contents
	fmt.Printf("disk number %s  is being wiped\n", drivenumber)
	wipestatus := wipedisk(drivenumber)
	fmt.Printf("result of disk wipe is %v\n", wipestatus)

	didinitwork := initializediskps(drivenumber)
	//	fmt.Printf("\nfunction call initializediskps(%s) returned: %d\n", drivenumber, didinitwork)
	if didinitwork != 0 {
		fmt.Printf("disk init result = %d for disk %v\n this should be a 0, so giving up", didinitwork, drivenumber)
		return
	}
	fmt.Println("disk has been re-initted, proceeding to partition and format it.")
	var cmdargs1 = fmt.Sprintf("new-partition -DiskNumber %v -assigndriveletter -usemaximumsize |  format-volume -filesystem ntfs -confirm:$false -newfilesystemlabel %v |Convertto-json", drivenumber, drivename)
	out, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		//		log.Print("error: %v \n" ,&err)
		if _, ok := err.(*exec.ExitError); ok {
			errmsg := fmt.Sprintf(`{"message": "Error occurred running \n%v \n. exiting."}`, cmdargs1)

			fmt.Printf("%v", errmsg)
			w.Write([]byte(errmsg))
			return
		}
	}
	w.Write(out)
}

//DelExports will an nfs export
func DelExports(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	pathParams := mux.Vars(r)
	exportname := pathParams["exportname"]

	//test for arguments that are missing or wrong
	sz := len(exportname)
	if sz == 0 {
		errmsg := fmt.Sprintln(`{"message": "Error no export name sent to delete. exiting."}`)
		fmt.Printf("%v", errmsg)
		w.Write([]byte(errmsg))
		return
	}
	fmt.Printf("exportname is %v\n", exportname)
	name, err := regexp.MatchString("^[a-zA-Z]+$", exportname)
	//fmt.Printf("m is %v", m)
	if err != nil || name != true {
		errmsg := `{"message": "export name not recognized or has bad characters, A-Z and a-z only, aka vnir or VNIR not /vnir. exiting."}`
		fmt.Printf("%v", errmsg)
		w.Write([]byte(errmsg))
		return
	}
	//check it was already exported before trying to delete the export
	checkifalreadyexported := checkifalreadyexportedname(exportname)

	if checkifalreadyexported == true {
		fmt.Println("exportname was found")
		//now delete it
		var cmdargs1 = fmt.Sprintf("Remove-NfsShare -Name \"%v\" -Confirm:$false", exportname)
		_, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
		if err != nil {
			fmt.Printf("ERROR when trying to remove export name %v\n", exportname)
			errmsg := fmt.Sprintf(`{"message": "ERROR when trying to remove export name %v"}`, exportname)
			fmt.Printf("%v", errmsg)
			w.Write([]byte(errmsg))
			return
		}
		//check its gone, then reply status
		checkexportisgone := checkifalreadyexportedname(exportname)
		if checkexportisgone == true {
			fmt.Printf("ERROR: export didnt delete, when asked to, name = %v\n", exportname)
			errmsg := fmt.Sprintf(`{"message": "ERROR export didnt delete, when asked to, name = %v"}`, exportname)
			fmt.Printf("%v", errmsg)
			w.Write([]byte(errmsg))
			return
		}
		if checkexportisgone == false {
			fmt.Printf("success: export %v was deleted\n", exportname)
			errmsg := fmt.Sprintf(`{"message": "SUCCESS: export %v was removed"}`, exportname)
			fmt.Printf("%v", errmsg)
			w.Write([]byte(errmsg))
			return
		}

	}
	errmsg := fmt.Sprintf(`{"message": "ERROR unknown error occured in function DelExports with exportname %v"}`, exportname)
	fmt.Printf("%v", errmsg)
	w.Write([]byte(errmsg))
}

//AddExports will create nfs exports
func AddExports(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	pathParams := mux.Vars(r)
	driveletter := pathParams["driveletter"]
	drivename := pathParams["drivename"]

	//test for arguments that are missing or wrong
	sz := len(driveletter)
	if sz == 0 {
		errmsg := fmt.Sprintln(`{"message": "Error no drive letter sent to export. exiting."}`)
		fmt.Printf("%v", errmsg)
		w.Write([]byte(errmsg))
		return
	}
	//match F: or J: etc
	checkedvar, err := regexp.MatchString("^[a-zA-Z]:$", driveletter)
	if err != nil {
		fmt.Printf("incorrect args passed as drive letter: %v", err)
		errmsg := fmt.Sprintf(`{"message": "Error incorrect args sent as drive letter: %v. exiting."}`, driveletter)
		fmt.Printf("%v", errmsg)
		w.Write([]byte(errmsg))
		return
	}
	if checkedvar == true {
		fmt.Printf("drive letter to export = %v\n", driveletter)
	}
	if checkedvar == false {
		fmt.Printf("incorrect args passed as drive letter: %v", err)
		errmsg := fmt.Sprintf(`{"message": "Error incorrect args sent as drive letter: %v. exiting."}`, driveletter)
		fmt.Printf("%v", errmsg)
		w.Write([]byte(errmsg))
		return
	}

	//ok now to do it, we should check if its already exported and if so, break that first, then export it.
	checkifalreadyexported := checkifalreadyexported(driveletter)

	if checkifalreadyexported == true {
		fmt.Println("disk was already exported, attempting to un-export first")
		unexport := unexport(driveletter)
		fmt.Printf("%t", unexport)
		fmt.Println(unexport)
	}

	//check to make sure the export is not already exported based on name
	checkifalreadyexportedname := checkifalreadyexportedname(drivename)

	if checkifalreadyexportedname == true {
		fmt.Println("an export with the name is already there, aborting request to export it.")
		errmsg := fmt.Sprintf(`{"message": "ERROR: export already exists with name %v"}`, drivename)
		fmt.Printf("%v", errmsg)
		w.Write([]byte(errmsg))
		return
	}

	//from here, we should be ok to add the export, if it was already there, its been fixed, may want to sleep a bit
	time.Sleep(3)

	//now attempt to export it
	createnfsexport := createnfsexport(driveletter, drivename)
	if createnfsexport == true {
		//fmt.Println("disk was successfully exported")
		//fix ntfs permission for nfs
		permfixed := fixntfspermsdwindowperms(driveletter)
		if permfixed == true {
			errmsg := fmt.Sprintln(`{"message": "Drive was successfully exported"}`)
			fmt.Printf("%v", errmsg)
			w.Write([]byte(errmsg))
		}
		if permfixed == false {
			errmsg := fmt.Sprintln(`{"message": "ERROR: Drive was exported, but permissions failed to apply correctly."}`)
			fmt.Printf("%v", errmsg)
			w.Write([]byte(errmsg))
		}
	}
}

func fixntfspermsdwindowperms(driveletter string) bool {
	if err := acl.Apply(
		driveletter,
		false,
		false,
		acl.GrantName(windows.GENERIC_ALL, "ANONYMOUS LOGON"),
	); err != nil {
		return false
		//panic(err)
	}
	return true
}

func createnfsexport(driveletter string, drivename string) bool {
	//var cmdargs1 = fmt.Sprintf("Remove-NfsShare -Path \"%v\\\" -Confirm:$false", driveletter)
	var cmdargs1 = fmt.Sprintf("New-NfsShare -Path \"%v\\\" -Name %v -AnonymousGid 65534 -AnonymousUid 65534 -EnableAnonymousAccess:$True -Authentication All -Permission readonly", driveletter, drivename)
	_, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		fmt.Printf("ERROR running createnfsexport %v with name %v", driveletter, drivename)
		return false
	}
	//check it was made
	checkifalreadyexported := checkifalreadyexported(driveletter)

	if checkifalreadyexported == true {
		fmt.Println("disk was exported successfully")
		return true
	}
	return false
}

func unexport(driveletter string) bool {
	//no drive sanity needed already checked
	var cmdargs1 = fmt.Sprintf("Remove-NfsShare -Path \"%v\\\" -Confirm:$false", driveletter)
	_, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		fmt.Printf("ERROR running unexport against %v", driveletter)
		return false
	}

	//no response to the command, so check it with the other function for success
	exportstillthere := checkifalreadyexported(driveletter)
	if exportstillthere == true {
		fmt.Printf("Failed to remove export = %v\n", driveletter)
		return false
	}
	if exportstillthere == false {
		fmt.Printf("Successfully removed export = %v\n", driveletter)
		return true
	}
	return false
}

func checkifalreadyexported(driveletter string) bool {
	//no drive sanity needed already checked
	var cmdargs1 = fmt.Sprintf("Get-NfsShare |where-object path -eq \"%v\\\" |measure-object| %% { $_.Count }", driveletter)
	out, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		fmt.Printf("ERROR running checkifalreadyexported against %v", driveletter)
		return false
	}
	fmt.Println(string(out))
	//48 = 0, 49 = 1
	if out[0] == byte(48) {
		fmt.Println("drive is NOT already exported")
		return false
	}
	if out[0] == byte(49) {
		fmt.Println("drive is already exported")
		return true
	}
	return false
}

func checkifalreadyexportedname(exportname string) bool {
	//no drive sanity needed already checked
	var cmdargs1 = fmt.Sprintf("Get-NfsShare |where-object Name -eq \"%v\" |measure-object| %% { $_.Count }", exportname)
	out, err := exec.Command("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", cmdargs1).Output()
	if err != nil {
		fmt.Printf("ERROR running checkifalreadyexportedname against %v", exportname)
		return false
	}
	fmt.Println(string(out))
	//48 = 0, 49 = 1
	if out[0] == byte(48) {
		fmt.Println("exportname is NOT already exported")
		return false
	}
	if out[0] == byte(49) {
		fmt.Println("exportname is already exported")
		return true
	}
	return false
}

var logger service.Logger

// Program structures.
//  Define Start and Stop methods.
type program struct {
	exit chan struct{}
}

func (p *program) Start(s service.Service) error {
	if service.Interactive() {
		logger.Info("Running in terminal.")
	} else {
		logger.Info("Running under service manager.")
	}
	p.exit = make(chan struct{})

	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}

func (p *program) run() error {

	//check if were looking for commands, to install or remove service, otherwise continue
	parseArgs()
	//logger.Infof("Starting EDOSapi %v.", service.Platform())
	isIntSess, _ := svc.IsAnInteractiveSession()

	if isIntSess {
		//fmt.Print("Interactive session detected, exiting, use debug to run manually if needed\n")
		os.Exit(1)
		return nil
	}

	mainroutine()
	return nil
}

func (p *program) Stop(s service.Service) error {
	// Any work in Stop should be quick, usually a few seconds at most.
	logger.Info("Stopping")
	close(p.exit)
	return nil
}

//GenRandomBytes make a random byte slice
func GenRandomBytes(size int) (blk []byte, err error) {
	blk = make([]byte, size)
	_, err = rand.Read(blk)
	return
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

//RandStringRunes sets the characters allowed in the hash
func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

//maketoken
func maketoken() string {
	//	seed, _ := GenRandomBytes(14)
	seed := RandStringRunes(14)
	tokenstr := string(seed)
	return tokenstr
}

//settoken
func settoken() string {
	//just to save code, and make it easy, make the password for them
	tokenstr := maketoken()
	//fmt.Printf("\ntoken %v\n", tokenstr)
	//no we have the token, store it in registry
	out, err := storetoken(tokenstr)
	if err != nil {
		log.Print(err)
		return "unable to set registry key, did you install the service yet?"
	}
	//read token from registry to be sure it saved
	//registrytoken := getregistrytoken()

	return out
}

//storetoken stores a string in the registry
func storetoken(tokenstr string) (string, error) {
	//fmt.Printf("\ntoken %v\n", tokenstr)
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\EDOSapi`, registry.WRITE)
	defer k.Close()
	if err != nil {
		log.Print(err)
		return "ERROR: Unable to open registry key, Run as Administrator?", err
	}

	key := "token"
	val := tokenstr
	if err = k.SetStringValue(key, val); err != nil {
		log.Print(err)
		return "ERROR: failed to set key, has the token been set yet?", err
	}
	out := fmt.Sprintf("Successfully set token %v\n\n---> Copy the following token, and set inside the Awapss Data Manager\n\n       %v\n\n", tokenstr, tokenstr)
	return out, err
}

//parseArgs will check command args to see if there is a service install or remove tag and take appropriate actions
func parseArgs() {
	isIntSess, err := svc.IsAnInteractiveSession()
	//if service.Interactive() {
	if !isIntSess {
		return
	}
	//fmt.Println("! interactive !")
	const svcName = "EDOSapi"

	if len(os.Args) < 2 {
		usage("no command specified")
	}

	cmd := strings.ToLower(os.Args[1])
	//err := nil
	switch cmd {
	//set the access token in the registry, only one allowed so over it, if its already there. then exit.
	case "settoken":
		result := settoken()
		log.Print(result)
		//os.Exit(1)
		return
	case "debug":
		//runService(svcName, true)
		mainroutine()
		return
	case "version":
		fmt.Printf("EDOS api Version %v\n", Version)
		return
	case "install":
		err := installService(svcName, "EDOSapi")
		if err != nil {
			log.Fatalf("failed to %s %s: %v", cmd, svcName, err)
		}
		fmt.Println("Service Installed")
	case "remove":
		err := removeService(svcName)
		if err != nil {
			log.Fatalf("failed to %s %s: %v", cmd, svcName, err)
		}
		fmt.Println("Service Removed")
	case "start":
		err := startService(svcName)
		if err != nil {
			log.Fatalf("failed to %s %s: %v", cmd, svcName, err)
		}
		fmt.Println("Service Started")
	case "stop":
		err := controlService(svcName, svc.Stop, svc.Stopped)
		if err != nil {
			log.Fatalf("failed to %s %s: %v", cmd, svcName, err)
		}
		fmt.Println("Service Stopped")
	case "pause":
		err := controlService(svcName, svc.Pause, svc.Paused)
		if err != nil {
			log.Fatalf("failed to %s %s: %v", cmd, svcName, err)
		}
		fmt.Println("Service Paused")
	case "continue":
		err := controlService(svcName, svc.Continue, svc.Running)
		if err != nil {
			log.Fatalf("failed to %s %s: %v", cmd, svcName, err)
		}
	default:
		usage(fmt.Sprintf("invalid command %s", cmd))
	}
	if err != nil {
		log.Fatalf("failed to %s %s: %v", cmd, svcName, err)
	}
	//}
}

func authentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Executing authentication")
		err := CheckToken(w, r)
		if err == nil {
			fmt.Println("ERROR detected")
			next.ServeHTTP(w, r) //`next.ServeHTTP(w, r)` will forward the request and response to next handler.
		}
	})
}

func getRealAddr(r *http.Request) string {
	remoteIP := ""
	// the default is the originating ip. but we try to find better options because this is almost
	// never the right IP
	if parts := strings.Split(r.RemoteAddr, ":"); len(parts) == 2 {
		remoteIP = parts[0]
	}
	// If we have a forwarded-for header, take the address from there
	//if xff := strings.Trim(r.Header.Get("X-Forwarded-For"), ","); len(xff) > 0 {
	//	addrs := strings.Split(xff, ",")
	//	lastFwd := addrs[len(addrs)-1]
	//	if ip := net.ParseIP(lastFwd); ip != nil {
	//		remoteIP = ip.String()
	//	}
	//	// parse X-Real-Ip header
	//} else if xri := r.Header.Get("X-Real-Ip"); len(xri) > 0 {
	//	if ip := net.ParseIP(xri); ip != nil {
	//		remoteIP = ip.String()
	//	}
	//}
	return remoteIP
}

func mainroutine() {
	gorm.NowFunc = func() time.Time {
		return time.Now().UTC()
	}

	//make the directory if its not there
	_, err := os.Stat("C:\\ProgramData\\edosAPI")
	if os.IsNotExist(err) {
		fmt.Println("Creating directory for storing the databases")
		errDir := os.MkdirAll("C:\\ProgramData\\edosAPI", 0755)
		if errDir != nil {
			log.Fatal(err)
		}
	}
	//get drive usage
	go func() {
		for {
			//collect drive capacity info every hour, keep for 30 rolling days
			drives := getalldrives()
			//loop over drives, to collect stats for each as they come and go.
			for _, s := range drives {
				CollectDiskstatsInBackground(s)
			}
			time.Sleep(3600 * time.Second)
		}
	}()
	//get diskio stats
	go func() {
		for {
			//collect diskio traffic every 10 seconds, keep for 2 hour
			drives := getalldrives()
			//loop over drives, to collect stats for each as they come and go.
			for _, s := range drives {
				CollectDiskIoStatsInBackground(s)
			}
			//delete stuff older then 1 hour
			//PurgeDbRecords(table, time reference, direction to keep)
			PurgeDbRecordsDiskios("-2 hours")
			time.Sleep(10 * time.Second)
		}
	}()
	//get network stats
	go func() {
		for {
			//collect network traffic every 10 seconds, keep for 2 hour
			CollectNetworkIoStatsInBackground()
			PurgeDbRecordsNetworkIoStats("-2 hours")
			time.Sleep(10 * time.Second)
		}
	}()

	//get cpu,mem stats
	go func() {
		for {
			//collect cpu and mem load every 10 seconds, keep for 2 hour
			CollectcpustatsInBackground()
			CollectmemstatsInBackground()
			//delete records older than 2 hours
			PurgeDbRecordsCPU("-2 hours")
			PurgeDbRecordsMEM("-2 hours")
			time.Sleep(10 * time.Second)
		}
	}()

	//init the web interface
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()

	api.HandleFunc("/dashboard", collectStatsHome)
	api.HandleFunc("/mem", collectStatsMEM).Methods(http.MethodGet)
	//gets memory stats from db, returns in json
	api.HandleFunc("/getmemstats", getmemstatsfromdb).Methods(http.MethodGet)
	api.HandleFunc("/cpu", collectStatsCPU).Methods(http.MethodGet)
	//gets cpu load history for 2 hours
	api.HandleFunc("/getcpustats", getcpustatsfromdb).Methods(http.MethodGet)
	api.HandleFunc("/disk/{driveletter}", collectStatsDISK).Methods(http.MethodGet)
	api.HandleFunc("/diskio", Iocounts).Methods(http.MethodGet)
	//get json output from the sql db for diskio
	api.HandleFunc("/getdiskio", getdiskstatsfromdb).Methods(http.MethodGet)
	//getjson output from the sql db for diskiostats
	api.HandleFunc("/getdiskiostats", getdiskiotatsfromdb).Methods(http.MethodGet)
	//show current network io -- no history here see getnetworkio for history
	api.HandleFunc("/networkio", nCounters).Methods(http.MethodGet)
	//get the json data to make a graph from the sqlite db for network
	api.HandleFunc("/getnetworkio", getnetworkstatsfromdb).Methods(http.MethodGet)
	//shows a list of url endpoints
	api.HandleFunc("/", listpaths).Methods(http.MethodGet)
	//general status function will be changed last
	api.HandleFunc("/status", status).Methods(http.MethodGet)
	//gets nfs export names and locations on edos
	api.HandleFunc("/getexports", getexports).Methods(http.MethodGet)
	//gets the drive status (total # and unconfigured good or bad, or configured good)
	api.HandleFunc("/getraiddrivestatus", getraiddrivestatus).Methods(http.MethodGet)
	//displays info on the raid unit if it imported
	api.HandleFunc("/getunitinfo", GetUnitInfo).Methods(http.MethodGet)
	//will get all the disks on controller 1, windows disks, not the raid data
	api.HandleFunc("/getpsdriveinfo", getpsdriveinfo).Methods(http.MethodGet)

	apiauthed := r.PathPrefix("/apiauthed/v1").Subrouter()
	//these ones are destructive, and need the key to operate
	apiauthed.Use(authentication)
	//will try to put a partition table on, format it with ntfs, and label it, (/vnir or /mwir?)
	apiauthed.HandleFunc("/initandpartitionandformatandnamedisk/{drivenumber}/{drivename}/{token}", initandpartitionandformatandnamedisknew).Methods(http.MethodGet)
	//will try to initialize a raw disk (just formed raids are raw, and have no partition table yet.)
	apiauthed.HandleFunc("/initializedisk/{drivenumber}/{token}", initializedisk).Methods(http.MethodGet)
	//attempt to delete a raid, assumes groups of four disks 0-3, 4-7, 8-11
	////////////////////incompelte function/////////////////////////////
	apiauthed.HandleFunc("/deleteraid/{ports}/{token}", deleteraid).Methods(http.MethodGet)
	//attempt to create a raid, assumes groups of four disks 0-3, 4-7, 8-11
	apiauthed.HandleFunc("/createraid/{ports}/{name}/{token}", createraidAPI).Methods(http.MethodGet)
	//restart the controller (rescans disks)
	apiauthed.HandleFunc("/rescandisks/{token}", rescandisks).Methods(http.MethodGet)
	//change disk from unconfigured bad, to good. no need for opposite action
	apiauthed.HandleFunc("/makeunconfiguredgood/{token}", MakeDrivesUnconfiguredGood).Methods(http.MethodGet)
	//try to create a new nfs export, driveletter will be the edos drive letter or "all"
	apiauthed.HandleFunc("/addexports/{driveletter}/{drivename}/{token}", AddExports).Methods(http.MethodGet)
	//try to remove an nfs export, by exported name
	apiauthed.HandleFunc("/removeexports/{exportname}/{token}", DelExports).Methods(http.MethodGet)
	//tries to import foreign array on controller
	apiauthed.HandleFunc("/importdrivepacks/{token}", importdrivepacks).Methods(http.MethodGet)
	apiauthed.HandleFunc("/exitnow/{token}", exitnow).Methods(http.MethodGet)
	//shows a list of url endpoints
	apiauthed.HandleFunc("/", listpathsauthed).Methods(http.MethodGet)
	//	api.HandleFunc("/getraidlogs", getraidlogs).Methods(http.MethodGet)
	//	api.HandleFunc("/getsomething", getexports).Methods(http.MethodGet)

	//the http listoner startup
	//	log.Fatal(http.ListenAndServe(":8000", r))
	ipandport := "0.0.0.0:8000"
	headers := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	methods := handlers.AllowedMethods([]string{"GET", "POST"})
	origins := handlers.AllowedOrigins([]string{"*"})
	//http.ListenAndServe()
	srv := &http.Server{
		Addr: ipandport,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      JustLocal(handlers.CORS(headers, methods, origins)(r)), // Pass our instance of gorilla/mux in.
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			//if err := srv.ListenAndServeTLS("server.crt", "server.key"); err != nil {
			log.Println(err)
			os.Exit(1)
		}
		fmt.Printf("API server listening on port %v started\n", ipandport)
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("shutting down")
	os.Exit(0)
}

func usage(errmsg string) {
	fmt.Fprintf(os.Stderr,
		"%s\n\n"+
			"usage: %s <command>\n"+
			"       where <command> is one of\n\n"+
			"		install  = Installs the service, starts service, sets access token \n"+
			"		version  = Displays Version \n"+
			"		remove   = removes the service, stops the service, removes token\n"+
			"		debug    = Runs the program in this shell (may need to stop service first)\n"+
			"		start    = Starts the service if installed\n"+
			"		stop	 = Stops the service if installed\n"+
			"		settoken = Sets the token required for the Awapss Data Manager to interface with the EDOS\n",
		errmsg, os.Args[0])
	os.Exit(2)
}

//JustLocal limit web traffic to local network
func JustLocal(handler http.Handler) http.Handler {
	var localsubnets []*net1.IPNet
	//get local ip
	ip, err := externalIP()
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Println(ip)

	//make the local ip found, become a .0 address with a /24 mask
	s := strings.Split(ip, ".")
	ipfixed := fmt.Sprintf("%v.%v.%v.0/24", s[0], s[1], s[2])
	fmt.Printf("API access granted to network %v\n", ipfixed)
	//define other local and non-routables
	localsubnetStrings := []string{"127.0.0.1/31", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", ipfixed}
	//localsubnetStrings := []string{"127.0.0.1/31", "172.16.0.0/12", "192.168.0.0/16", ipfixed}

	for _, netstrings := range localsubnetStrings {
		_, n, _ := net1.ParseCIDR(netstrings)
		localsubnets = append(localsubnets, n)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//fmt.Println(r.RemoteAddr)
		remoteip := net1.ParseIP(strings.Split(r.RemoteAddr, ":")[0])
		//fmt.Printf("page requested by: %v", remoteip)

		local := false
		for _, localsubnet := range localsubnets {
			//fmt.Println(localsubnet, remoteip)
			if localsubnet.Contains(remoteip) {
				fmt.Println("local ip network request matched")
				local = true
				break
			}
		}
		if !local {
			fmt.Printf("Un-authorized network request from %v\n", remoteip)
			http.Error(w, "Un-authorized", 403)
			return
		}
		handler.ServeHTTP(w, r)
		return
	})
}

func externalIP() (string, error) {
	ifaces, err := net1.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net1.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net1.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net1.IP
			switch v := addr.(type) {
			case *net1.IPNet:
				ip = v.IP
			case *net1.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

//CheckToken compares the token to the registry token for api requests
func CheckToken(w http.ResponseWriter, r *http.Request) error {

	Params := mux.Vars(r)
	//fmt.Printf
	token := Params["token"]
	var err error
	if len(token) == 0 {
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			//w.WriteHeader(http.StatusOK)
			errmsg := fmt.Sprint(`{"ERROR": "Sorry, set token first."}`)

			http.Error(w, errmsg, http.StatusForbidden)
			return errors.New(errmsg)
		}
	}
	//check the registry for the token to see if they match or not. if not, err and exit.
	tokenmatchbool, tokenmatchstr, err := CheckTokenmatch(token)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		errmsg := fmt.Sprintf(`{"ERROR": "%v"}`, tokenmatchstr)
		http.Error(w, errmsg, http.StatusForbidden)

		return errors.New(errmsg)
	}

	if tokenmatchbool == false {
		w.Header().Set("Content-Type", "application/json")
		//w.WriteHeader(http.StatusOK)
		errmsg := fmt.Sprintf(`{"ERROR": "%v"}`, tokenmatchstr)
		http.Error(w, errmsg, http.StatusForbidden)

		return errors.New(errmsg)
	}
	// if were here, its good and passed tests.
	if tokenmatchbool == true {
		//w.Header().Set("Content-Type", "application/json")
		//w.WriteHeader(http.StatusOK)
		//errmsg := fmt.Sprintf(`{"ERROR": "%v"}`, tokenmatchstr)
		//http.Error(w, errmsg, http.StatusForbidden)
		return nil
	}
	return err
}

//CheckTokenmatch checks the registry value against the string, true for match, false for no match, err for no key at all
func CheckTokenmatch(token string) (bool, string, error) {
	//fmt.Printf("\ntoken %v\n", tokenstr)
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\EDOSapi`, registry.QUERY_VALUE)
	defer k.Close()

	if err != nil {
		log.Print(err)
		msg := "ERROR: Unable to READ registry key, has it been set?"
		return false, msg, errors.New(msg)
	}

	keyvalue, _, err := k.GetStringValue("token")
	if err != nil {
		log.Print(err)
		msg := "ERROR: failed to Read key"
		return false, msg, errors.New(msg)
	}
	fmt.Printf("key is %v\nToken is %v\n", keyvalue, token)

	//out := fmt.Sprintf("Successfully set token %v\n\n---> Copy the following token, and set inside the Awapss Data Manager\n\n       %v\n\n", tokenstr, tokenstr)
	if keyvalue == token {
		fmt.Println("tokens match, next..")
		return true, "tokens match, next..", nil
	}
	if keyvalue != token {
		fmt.Println("tokens do not match, aborting")
		msg := "tokens do not match, arborting"
		return false, msg, errors.New(msg)
	}
	return false, "Unknown error occurred", err
}

func main() {
	svcConfig := &service.Config{
		Name:        "EDOSapi",
		DisplayName: "EDOSapi",
		Description: "API for data manager",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	logger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}
	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}
