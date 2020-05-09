package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

const csvFileDay1 string = "no-dups-day1.csv"
const csvFileDay2 string = "no-dups-day2.csv"
const csvFileDay3 string = "no-dups-day3.csv"

const durationsDay1Csv = "durations-day1.csv"
const durationsDay2Csv = "durations-day2.csv"
const durationsDay3Csv = "durations-day3.csv"

const conactsDay1Csv string = "contacts-day1.csv"
const conactsDay2Csv string = "contacts-day2.csv"
const conactsDay3Csv string = "contacts-day3.csv"

const idsDay1Csv string = "mac-to-id-data-day1.csv"
const idsDay2Csv string = "mac-to-id-data-day2.csv"
const idsDay3Csv string = "mac-to-id-data-day3.csv"

const macAddr int = 0
const loc int = 1
const timestamp int = 2

// subtract 3 hours from the timestamp because while
// converting the timestamp to unix time, it does so
// in regards to the local time which is to utc + 3
const utc3Hours = 3 * 60 * 60

type EventRecord struct {
	nodeId int
	start  int64
	end    int64
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func createContacts(myMap map[string][]EventRecord, csvContactsFile string, day int, wg *sync.WaitGroup) {
	defer wg.Done()

	var contacts, nodeContacts int64
	var nodeContactLimit int64 = 150

	file, err := os.OpenFile(csvContactsFile, os.O_CREATE|os.O_WRONLY, 0660)
	if err != nil {
		panic(err)
	}
	writer := csv.NewWriter(file)

	err = writer.Write([]string{
		"id1", "id2", "tstart", "tend", "location",
	})
	if err != nil {
		panic(err)
	}
	for key, mySlice := range myMap {
		contacts = 0
		for i := 0; i < len(mySlice)-1; i++ {
			nodeContacts = 0
			nodeId := mySlice[i].nodeId
			for j := i + 1; j < len(mySlice); j++ {
				if nodeContacts >= nodeContactLimit {
					break
				}
				// this condition is necessary because a node might return
				// multiple times during a day at a certain location
				if nodeId == mySlice[j].nodeId {
					continue
				}
				// if node A arrives at the location X
				// and node B was already there, create a
				// contact opportunity
				// take into account that my start time
				// has to be before the node B leaves
				if (mySlice[i].start > mySlice[j].start) &&
					(mySlice[i].start < mySlice[j].end) {
					nodeContacts++
					contacts++
					err = writer.Write([]string{
						strconv.Itoa(nodeId),
						strconv.Itoa(mySlice[j].nodeId),
						strconv.FormatInt(mySlice[i].start, 10),
						strconv.FormatInt(min(mySlice[i].end, mySlice[j].end), 10),
						key})
					if err != nil {
						panic(err)
					}
				}
			}
		}
		fmt.Println(day, "no_nodes", strconv.Itoa(len(mySlice)), "location",
			key, "contacts", strconv.FormatInt(contacts, 10))
	}

	writer.Flush()
	file.Close()
}

func macsToIds(utcPrt *int64, numbPtr *int, csvInFile string, csvMacToIdFile string, csvContactsFile string, dayMap map[string][]EventRecord, wg *sync.WaitGroup) {
	defer wg.Done()

	var result [][]string

	// open csv input file
	file, err := os.OpenFile(csvInFile, os.O_RDWR, 0660)
	if err != nil {
		panic(err)
	}
	reader := csv.NewReader(file)
	// read all records
	result, _ = reader.ReadAll()
	file.Close()

	// open file to write node ids
	idsFile, err := os.OpenFile(csvMacToIdFile, os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		panic(err)
	}
	writer := csv.NewWriter(idsFile)
	err = writer.Write([]string{
		"id", "location", "timestamp",
	})

	nodeID := 0
	strNodeID := strconv.Itoa(nodeID)
	currentMac := result[1][macAddr]
	result[1][macAddr] = strNodeID
	// convert the timestamp
	layout := "2006-01-02 15:04:05"
	t, err := time.Parse(layout, result[1][timestamp])
	day := t.Day()
	fmt.Println(day)
	location := result[1][loc]
	// convert the timestamp to unix time
	startTime, prevTime := t.Unix()-*utcPrt, t.Unix()-*utcPrt
	fmt.Println(startTime)

	// write id to file in order to count nodes for simulation
	err = writer.Write([]string{
		strNodeID,
		result[1][loc],
		result[1][timestamp],
	})
	if err != nil {
		panic(err)
	}

	for i := 2; i < len(result); i++ {
		// if nodeID >= *numbPtr {
		// 	fmt.Println("nodeId = ", nodeID, "break")
		// 	break
		// }

		t, err = time.Parse(layout, result[i][timestamp])
		// set the nodeID
		if result[i][macAddr] == currentMac {
			// write id to file in order to count nodes for simulation
			err := writer.Write([]string{
				strNodeID,
				result[i][loc],
				result[i][timestamp],
			})
			if err != nil {
				panic(err)
			}
			/*
			 * if the day and the location have not changed,
			 * contiune to look for the moment the node left that
			 * location
			 */
			// fmt.Printf("node %d loc1 %s loc2 %s day1 %d day2 %d\n", nodeID, location, result[i][loc], day, t.Day())
			if result[i][loc] == location {
				prevTime = t.Unix() - *utcPrt
				continue
			} else {
				/*
				 * OUTLIER REMOVAL the node was spotted only once at the scene
				 * and that information is useless, thus we
				 * prepare the  variables for the next day or
				 * location
				 */
				if prevTime == startTime {
					location = result[i][loc]
					startTime, prevTime = t.Unix()-*utcPrt, t.Unix()-*utcPrt
					continue
				} else {
					event := EventRecord{
						nodeID,
						startTime,
						prevTime,
					}

					// fmt.Printf("ADD node %d loc %s day %d startT %d endT %d\n", nodeID, location, day, startTime, prevTime)
					dayMap[location] = append(dayMap[location], event)

					location = result[i][loc]
					startTime, prevTime = t.Unix()-*utcPrt, t.Unix()-*utcPrt
				}
			}
		} else {
			currentMac = result[i][macAddr]
			nodeID++
			strNodeID = strconv.Itoa(nodeID)

			// write id to file in order to count nodes for simulation
			err := writer.Write([]string{
				strNodeID,
				result[i][loc],
				result[i][timestamp],
			})
			if err != nil {
				panic(err)
			}

			if prevTime == startTime {
				location = result[i][loc]
				startTime, prevTime = t.Unix()-*utcPrt, t.Unix()-*utcPrt
				continue
			} else {
				lastID := nodeID - 1
				event := EventRecord{
					lastID,
					startTime,
					prevTime,
				}

				// fmt.Printf("ADD node %d loc %s day %d startT %d endT %d\n",nodeID, location, day, startTime, prevTime)
				dayMap[location] = append(dayMap[location], event)

				// set the variables for the new node
				location = result[i][loc]
				startTime, prevTime = t.Unix()-*utcPrt, t.Unix()-*utcPrt
			}
		}
	}

	fmt.Println(day, "No of nodes = ", nodeID)

	writer.Flush()
	idsFile.Close()
}

func computeTimePerDay(dayMap map[string][]EventRecord, csvFile string, durations map[int]int64, blacklist map[int]bool, wg *sync.WaitGroup) {
	defer wg.Done()

	for _, mySlice := range dayMap {
		for i := 0; i < len(mySlice)-1; i++ {
			dt := (mySlice[i].end - mySlice[i].start) / 60
			// test if there is an enty for nodeId
			_, ok := durations[mySlice[i].nodeId]
			if ok {
				durations[mySlice[i].nodeId] += dt
			} else {
				durations[mySlice[i].nodeId] = dt
			}
		}
	}

	// create a blacklist of nodes who spent less than 60 min at the festival
	for key, val := range durations {
		if val < 60 {
			blacklist[key] = true
		}
	}

	// file, err := os.OpenFile(csvFile, os.O_CREATE|os.O_WRONLY, 0660)
	// if err != nil {
	// 	panic(err)
	// }
	// writer := csv.NewWriter(file)

	// writer.Write([]string{"id", "duration"})
	// for key, value := range durations {
	// 	writer.Write([]string{strconv.Itoa(key), strconv.FormatInt(value, 10)})
	// }

	// writer.Flush()
	// file.Close()
}

func main() {
	numbPtr := flag.Int("nodes", 1000, "the number of nodes for mobemu simulation")
	utcPrt := flag.Int64("utcDiff", utc3Hours, "the number to subtract in order to obtain the utc time")
	flag.Parse()
	fmt.Println("nodes =", *numbPtr)

	day1Map := make(map[string][]EventRecord)
	day2Map := make(map[string][]EventRecord)
	day3Map := make(map[string][]EventRecord)

	// use a WaitGroup to sync all 3 goroutines
	var wg sync.WaitGroup
	wg.Add(1)
	go macsToIds(utcPrt, numbPtr, csvFileDay1, idsDay1Csv, conactsDay1Csv, day1Map, &wg)
	wg.Add(1)
	go macsToIds(utcPrt, numbPtr, csvFileDay2, idsDay2Csv, conactsDay2Csv, day2Map, &wg)
	wg.Add(1)
	go macsToIds(utcPrt, numbPtr, csvFileDay3, idsDay3Csv, conactsDay3Csv, day3Map, &wg)

	wg.Wait()

	durations1 := make(map[int]int64)
	durations2 := make(map[int]int64)
	durations3 := make(map[int]int64)
	// blacklists
	blacklist1 := make(map[int]bool)
	blacklist2 := make(map[int]bool)
	blacklist3 := make(map[int]bool)

	// compute the total time a node spent at the festival per day
	wg.Add(1)
	go computeTimePerDay(day1Map, durationsDay1Csv, durations1, blacklist1, &wg)
	wg.Add(1)
	go computeTimePerDay(day2Map, durationsDay2Csv, durations2, blacklist2, &wg)
	wg.Add(1)
	go computeTimePerDay(day3Map, durationsDay3Csv, durations3, blacklist3, &wg)

	wg.Wait()

}
