/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package cgroups

// import (
// 	"bufio"
// 	"fmt"
// 	"io/ioutil"
// 	"os"
// 	"path/filepath"
// 	"strconv"
// 	"strings"

// 	v1 "github.com/containerd/cgroups/stats/v1"
// )

// // NewBlkio returns a Blkio controller given the root folder of cgroups.
// // It may optionally accept other configuration options, such as ProcRoot(path)
// func NewBlkio(root string, options ...func(controller *blkioController)) *blkioController {
// 	ctrl := &blkioController{
// 		root:     filepath.Join(root, string(Blkio)),
// 		procRoot: "/proc",
// 	}
// 	for _, opt := range options {
// 		opt(ctrl)
// 	}
// 	return ctrl
// }

// // ProcRoot overrides the default location of the "/proc" filesystem
// func ProcRoot(path string) func(controller *blkioController) {
// 	return func(c *blkioController) {
// 		c.procRoot = path
// 	}
// }

// type blkioController struct {
// 	root     string
// 	procRoot string
// }

// func (b *blkioController) Name() Name {
// 	return Blkio
// }

// func (b *blkioController) Path(path string) string {
// 	return filepath.Join(b.root, path)
// }

// func (b *blkioController) Create(path string, resources *Resources) error {
// 	if err := os.MkdirAll(b.Path(path), defaultDirPerm); err != nil {
// 		return err
// 	}
// 	if resources.BlockIO == nil {
// 		return nil
// 	}
// 	for _, t := range createBlkioSettings(resources.BlockIO) {
// 		if t.value != nil {
// 			if err := ioutil.WriteFile(
// 				filepath.Join(b.Path(path), fmt.Sprintf("blkio.%s", t.name)),
// 				t.format(t.value),
// 				defaultFilePerm,
// 			); err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

// func (b *blkioController) Update(path string, resources *Resources) error {
// 	return b.Create(path, resources)
// }

// func (b *blkioController) Stat(path string, stats *Stats) error {
// 	stats.Blkio = BlkIOStat{}
// 	settings := []blkioStatSettings{
// 		{
// 			name:  "throttle.io_serviced",
// 			entry: &stats.Blkio.IoServicedRecursive,
// 		},
// 		{
// 			name:  "throttle.io_service_bytes",
// 			entry: &stats.Blkio.IoServiceBytesRecursive,
// 		},
// 	}
// 	// Try to read CFQ stats available on all CFQ enabled kernels first
// 	if _, err := os.Lstat(filepath.Join(b.Path(path), fmt.Sprintf("blkio.io_serviced_recursive"))); err == nil {
// 		settings = []blkioStatSettings{}
// 		settings = append(settings,
// 			blkioStatSettings{
// 				name:  "sectors_recursive",
// 				entry: &stats.Blkio.SectorsRecursive,
// 			},
// 			blkioStatSettings{
// 				name:  "io_service_bytes_recursive",
// 				entry: &stats.Blkio.IoServiceBytesRecursive,
// 			},
// 			blkioStatSettings{
// 				name:  "io_serviced_recursive",
// 				entry: &stats.Blkio.IoServicedRecursive,
// 			},
// 			blkioStatSettings{
// 				name:  "io_queued_recursive",
// 				entry: &stats.Blkio.IoQueuedRecursive,
// 			},
// 			blkioStatSettings{
// 				name:  "io_service_time_recursive",
// 				entry: &stats.Blkio.IoServiceTimeRecursive,
// 			},
// 			blkioStatSettings{
// 				name:  "io_wait_time_recursive",
// 				entry: &stats.Blkio.IoWaitTimeRecursive,
// 			},
// 			blkioStatSettings{
// 				name:  "io_merged_recursive",
// 				entry: &stats.Blkio.IoMergedRecursive,
// 			},
// 			blkioStatSettings{
// 				name:  "time_recursive",
// 				entry: &stats.Blkio.IoTimeRecursive,
// 			},
// 		)
// 	}
// 	for _, t := range settings {
// 		if err := b.readEntry(devices, path, t.name, t.entry); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// func (b *blkioController) readEntry(devices map[deviceKey]string, path, name string, entry *[]*v1.BlkIOEntry) error {
// 	f, err := os.Open(filepath.Join(b.Path(path), fmt.Sprintf("blkio.%s", name)))
// 	if err != nil {
// 		return err
// 	}
// 	defer f.Close()
// 	sc := bufio.NewScanner(f)
// 	for sc.Scan() {
// 		if err := sc.Err(); err != nil {
// 			return err
// 		}
// 		// format: dev type amount
// 		fields := strings.FieldsFunc(sc.Text(), splitBlkIOStatLine)
// 		if len(fields) < 3 {
// 			if len(fields) == 2 && fields[0] == "Total" {
// 				// skip total line
// 				continue
// 			} else {
// 				return fmt.Errorf("Invalid line found while parsing %s: %s", path, sc.Text())
// 			}
// 		}
// 		major, err := strconv.ParseUint(fields[0], 10, 64)
// 		if err != nil {
// 			return err
// 		}
// 		minor, err := strconv.ParseUint(fields[1], 10, 64)
// 		if err != nil {
// 			return err
// 		}
// 		op := ""
// 		valueField := 2
// 		if len(fields) == 4 {
// 			op = fields[2]
// 			valueField = 3
// 		}
// 		v, err := strconv.ParseUint(fields[valueField], 10, 64)
// 		if err != nil {
// 			return err
// 		}
// 		*entry = append(*entry, &v1.BlkIOEntry{
// 			Device: devices[deviceKey{major, minor}],
// 			Major:  major,
// 			Minor:  minor,
// 			Op:     op,
// 			Value:  v,
// 		})
// 	}
// 	return nil
// }

// func createBlkioSettings(blkio *BlockIOResource) []blkioSettings {
// 	settings := []blkioSettings{}

// 	if blkio.Weight != nil {
// 		settings = append(settings,
// 			blkioSettings{
// 				name:   "weight",
// 				value:  blkio.Weight,
// 				format: uintf,
// 			})
// 	}
// 	if blkio.LeafWeight != nil {
// 		settings = append(settings,
// 			blkioSettings{
// 				name:   "leaf_weight",
// 				value:  blkio.LeafWeight,
// 				format: uintf,
// 			})
// 	}
// 	return settings
// }

// type blkioSettings struct {
// 	name   string
// 	value  interface{}
// 	format func(v interface{}) []byte
// }

// type blkioStatSettings struct {
// 	name  string
// 	entry *[]*BlkIOEntry
// }

// func uintf(v interface{}) []byte {
// 	return []byte(strconv.FormatUint(uint64(*v.(*uint16)), 10))
// }

// func splitBlkIOStatLine(r rune) bool {
// 	return r == ' ' || r == ':'
// }
