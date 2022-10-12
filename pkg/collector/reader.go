/*
Copyright 2021.

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

package collector

import (
	"bytes"
	"encoding/binary"
	"math"
	"sync"
	"time"
	"unsafe"

	"github.com/sustainable-computing-io/kepler/pkg/attacher"
	"github.com/sustainable-computing-io/kepler/pkg/cgroup"
	"github.com/sustainable-computing-io/kepler/pkg/model"
	"github.com/sustainable-computing-io/kepler/pkg/podlister"
	"github.com/sustainable-computing-io/kepler/pkg/power/acpi"
	"github.com/sustainable-computing-io/kepler/pkg/power/gpu"
	"github.com/sustainable-computing-io/kepler/pkg/power/rapl"
	"github.com/sustainable-computing-io/kepler/pkg/power/rapl/source"

	"k8s.io/klog/v2"
)

// #define CPU_VECTOR_SIZE 128
import "C"

// TODO in sync with bpf program
type CgroupTime struct {
	CGroupPID      uint64
	PID            uint64
	ProcessRunTime uint64
	CPUCycles      uint64
	CPUInstr       uint64
	CacheMisses    uint64
	Command        [16]byte
	CPUTime        [C.CPU_VECTOR_SIZE]uint16
}

const (
	samplePeriodSec = 3
	samplePeriod    = samplePeriodSec * 1000 * time.Millisecond

	maxInactivePods = 10
)

var (
	// latest read energy
	sensorEnergy = map[string]float64{}
	pkgEnergy    = map[int]source.RAPLEnergy{}
	// latest process energy
	podEnergy  = map[string]*PodEnergy{}
	nodeEnergy = NewNodeEnergy()

	cpuFrequency = map[int32]uint64{}

	acpiPowerMeter = acpi.NewACPIPowerMeter()
	lock           sync.Mutex

	systemProcessName      = podlister.GetSystemProcessName()
	systemProcessNamespace = podlister.GetSystemProcessNamespace()
)

func init() {
	pods, err := podlister.Init()
	if err != nil {
		klog.V(5).Infoln(err)
		return
	}
	for i := 0; i < len(*pods); i++ {
		podName := (*pods)[i].Name
		podNamespace := (*pods)[i].Namespace
		podID := string((*pods)[i].ObjectMeta.UID)
		podEnergy[podID] = NewPodEnergy(podName, podNamespace)
	}
}

// readEnergy reads sensor/pkg energies in mJ
func (c *Collector) readEnergy() {
	sensorEnergy, _ = acpiPowerMeter.GetEnergyFromHost()
	pkgEnergy = rapl.GetRAPLEnergy()
}

// resetCurrValue reset existing podEnergy previous curr value
func (c *Collector) resetCurrValue() {
	for _, v := range podEnergy {
		v.ResetCurr()
	}
	nodeEnergy.ResetCurr()
}

// resetBPFTables reset BPF module's tables
func (c *Collector) resetBPFTables() {
	c.modules.Table.DeleteAll()
	c.modules.TimeTable.DeleteAll()
}

// readBPFEvent reads BPF event and maps between pid/cgroupid and container/pod
// initializes podEnergy component if not exists
// adds stats from BPF events (CPU time, available HW counters)
func (c *Collector) readBPFEvent() (pidPodID map[uint32]string, containerIDPodID map[string]string) {
	pidPodID = make(map[uint32]string)
	containerIDPodID = make(map[string]string)
	if c.modules == nil {
		return nil, nil
	}
	foundPod := make(map[string]bool)
	var ct CgroupTime
	for it := c.modules.Table.Iter(); it.Next(); {
		data := it.Leaf()
		err := binary.Read(bytes.NewBuffer(data), binary.LittleEndian, &ct)
		if err != nil {
			klog.V(5).Infof("failed to decode received data: %v", err)
			continue
		}
		comm := (*C.char)(unsafe.Pointer(&ct.Command))

		podID, err := podlister.GetPodID(ct.CGroupPID, ct.PID)
		if err != nil || podID == "" {
			klog.V(5).Infof("failed to resolve pod ID for cGroup ID %v: %v, set podName=%s", ct.CGroupPID, err, systemProcessName)
			podID = "system"
		}
		podName, err := podlister.GetPodName(ct.CGroupPID, ct.PID)

		if err != nil {
			klog.V(5).Infof("failed to resolve pod name for cGroup ID %v: %v, set podName=%s", ct.CGroupPID, err, systemProcessName)
			podName = systemProcessName
		}
		if _, ok := podEnergy[podID]; !ok {
			// new pod
			var podNamespace string
			if podName == systemProcessName {
				podNamespace = systemProcessNamespace
			} else {
				podNamespace, err = podlister.GetPodNameSpace(ct.CGroupPID, ct.PID)
				if err != nil {
					klog.V(5).Infof("failed to find namespace for cGroup ID %v: %v", ct.CGroupPID, err)
					podNamespace = "unknown"
				}
			}
			podEnergy[podID] = NewPodEnergy(podName, podNamespace)
		}
		foundPod[podID] = true

		podEnergy[podID].SetLatestProcess(ct.CGroupPID, ct.PID, C.GoString(comm))

		var activeCPUs []int32
		var avgFreq float64
		var totalCPUTime uint64
		if attacher.EnableCPUFreq {
			avgFreq, totalCPUTime, activeCPUs = getAVGCPUFreqAndTotalCPUTime(cpuFrequency, &ct.CPUTime)
			podEnergy[podID].AvgCPUFreq = avgFreq
		} else {
			totalCPUTime = ct.ProcessRunTime
			activeCPUs = getActiveCPUs(&ct.CPUTime)
		}

		for _, cpu := range activeCPUs {
			podEnergy[podID].CurrCPUTimePerCPU[uint32(cpu)] += uint64(ct.CPUTime[cpu])
		}

		if err = podEnergy[podID].CPUTime.AddNewCurr(totalCPUTime); err != nil {
			klog.V(5).Infoln(err)
		}

		for _, counterKey := range availableCounters {
			var val uint64
			switch counterKey {
			case attacher.CPUCycleLable:
				val = ct.CPUCycles
			case attacher.CPUInstructionLabel:
				val = ct.CPUInstr
			case attacher.CacheMissLabel:
				val = ct.CacheMisses
			default:
				val = 0
			}
			if err = podEnergy[podID].CounterStats[counterKey].AddNewCurr(val); err != nil {
				klog.V(5).Infoln(err)
			}
		}

		podEnergy[podID].CurrProcesses++
		containerID, err := podlister.GetContainerID(ct.CGroupPID, ct.PID)
		if err != nil {
			klog.V(5).Infoln(err)
		}
		// first-time found container (should not include non-container event)
		if _, found := containerIDPodID[containerID]; !found && podName != systemProcessName {
			containerIDPodID[containerID] = podID
			// TO-DO: move to container-level section
			rBytes, wBytes, disks, err := podlister.ReadCgroupIOStat(ct.CGroupPID, ct.PID)

			if err == nil {
				if disks > podEnergy[podID].Disks {
					podEnergy[podID].Disks = disks
				}
				podEnergy[podID].BytesRead.AddStat(containerID, rBytes)
				podEnergy[podID].BytesWrite.AddStat(containerID, wBytes)
			}
		}
		pid := uint32(ct.PID)
		if _, found := pidPodID[pid]; !found {
			pidPodID[pid] = podID
		}
	}
	c.resetBPFTables()
	handleInactivePods(foundPod)
	return pidPodID, containerIDPodID
}

// readCgroup adds container-level cgroup data
func (c *Collector) readCgroup(containerIDPodID map[string]string) {
	for containerID, podID := range containerIDPodID {
		cgroup.TryInitStatReaders(containerID)
		cgroupFSStandardStats := cgroup.GetStandardStat(containerID)
		for cgroupFSKey, cgroupFSValue := range cgroupFSStandardStats {
			readVal := cgroupFSValue.(uint64)
			if _, ok := podEnergy[podID].CgroupFSStats[cgroupFSKey]; ok {
				podEnergy[podID].CgroupFSStats[cgroupFSKey].AddStat(containerID, readVal)
			}
		}
	}
}

// readKubelet adds kubelet data (resident mem)
func (c *Collector) readKubelet() {
	if len(availableKubeletMetrics) == 2 {
		podCPU, podMem, _, _, _ := podlister.GetPodMetrics()
		klog.V(5).Infof("Kubelet Read: %v, %v\n", podCPU, podMem)
		for _, v := range podEnergy {
			k := v.Namespace + "/" + v.PodName
			readCPU := uint64(podCPU[k])
			readMem := uint64(podMem[k])
			cpuMetricName := availableKubeletMetrics[0]
			memMetricName := availableKubeletMetrics[1]
			if err := v.KubeletStats[cpuMetricName].SetNewAggr(readCPU); err != nil {
				klog.V(5).Infoln(err)
			}
			if err := v.KubeletStats[memMetricName].SetNewAggr(readMem); err != nil {
				klog.V(5).Infoln(err)
			}
		}
	}
}

func (c *Collector) reader() {
	ticker := time.NewTicker(samplePeriod)
	go func() {
		_ = gpu.GetGpuEnergy() // reset power usage counter
		c.resetBPFTables()
		c.readEnergy()
		nodeEnergy.SetValues(sensorEnergy, pkgEnergy, 0, map[string]float64{}) // set initial energy
		acpiPowerMeter.Run()
		for {
			// wait ticker
			<-ticker.C

			lock.Lock()
			c.resetCurrValue()
			var coreNDelta, dramNDelta, uncoreNDelta, pkgNDelta, gpuNDelta []float64
			// read node-level settings (frequency)
			cpuFrequency = acpiPowerMeter.GetCPUCoreFrequency()
			// read pod metrics
			pidPodID, containerIDPodID := c.readBPFEvent()
			c.readCgroup(containerIDPodID)
			c.readKubelet()
			// convert to pod metrics to array
			var podMetricValues [][]float64
			var podIDList []string
			for podID, v := range podEnergy {
				values := v.ToEstimatorValues()
				podMetricValues = append(podMetricValues, values)
				podIDList = append(podIDList, podID)
			}
			// TO-DO: handle metrics read by GPU device in the same way as the other usage metrics
			// read gpu power
			gpuPerPid, _ := gpu.GetCurrGpuEnergyPerPid() // power not energy
			podGPUDelta := make(map[string]float64)
			for pid, podID := range pidPodID {
				gpuPower := gpuPerPid[pid]
				if _, found := podGPUDelta[podID]; !found {
					podGPUDelta[podID] = 0
				} else {
					podGPUDelta[podID] += gpuPower
				}
			}
			for _, podID := range podIDList {
				gpuNDelta = append(gpuNDelta, podGPUDelta[podID])
			}
			// read and compute power (energy delta)
			var totalCorePower, totalDRAMPower, totalUncorePower, totalPkgPower, totalGPUPower uint64

			for _, val := range gpuNDelta {
				totalGPUPower += uint64(val)
			}
			c.readEnergy()
			sumUsage := model.GetSumUsageMap(metricNames, podMetricValues)
			nodeEnergy.SetValues(sensorEnergy, pkgEnergy, totalGPUPower, sumUsage)
			for pkgIDKey, pkgStat := range nodeEnergy.EnergyInPkg.Stat {
				coreDelta, dramDelta, uncoreDelta := nodeEnergy.GetCurrEnergyPerpkgID(pkgIDKey)
				pkgDelta := pkgStat.Curr
				coreNDelta = append(coreNDelta, float64(coreDelta))
				dramNDelta = append(dramNDelta, float64(dramDelta))
				uncoreNDelta = append(uncoreNDelta, float64(uncoreDelta))
				pkgNDelta = append(pkgNDelta, float64(pkgDelta))
				totalCorePower += coreDelta
				totalDRAMPower += dramDelta
				totalUncorePower += uncoreDelta
				totalPkgPower += pkgDelta
			}
			// get power from usage ratio
			podCore, podDRAM, podUncore, podPkg := model.GetPowerFromUsageRatio(podMetricValues, totalCorePower, totalDRAMPower, totalUncorePower, totalPkgPower, sumUsage)
			// get dynamic power from usage metrics
			podDynamicPower := model.GetDynamicPower(metricNames, podMetricValues, coreNDelta, dramNDelta, uncoreNDelta, pkgNDelta, gpuNDelta)
			// get other energy - divide equally
			podOther := uint64(0)
			podCount := len(podIDList)
			if podCount > 0 {
				podOther = nodeEnergy.EnergyInOther / uint64(podCount)
			}

			// set pod energy
			for i, podID := range podIDList {
				if err := podEnergy[podID].EnergyInCore.AddNewCurr(podCore[i]); err != nil {
					klog.V(5).Infoln(err)
				}
				if err := podEnergy[podID].EnergyInDRAM.AddNewCurr(podDRAM[i]); err != nil {
					klog.V(5).Infoln(err)
				}
				if err := podEnergy[podID].EnergyInUncore.AddNewCurr(podUncore[i]); err != nil {
					klog.V(5).Infoln(err)
				}
				if err := podEnergy[podID].EnergyInPkg.AddNewCurr(podPkg[i]); err != nil {
					klog.V(5).Infoln(err)
				}
				podGPU := uint64(math.Ceil(podGPUDelta[podID]))
				if err := podEnergy[podID].EnergyInGPU.AddNewCurr(podGPU); err != nil {
					klog.V(5).Infoln(err)
				}
				if err := podEnergy[podID].EnergyInOther.AddNewCurr(podOther); err != nil {
					klog.V(5).Infoln(err)
				}
			}
			if len(podDynamicPower) != 0 {
				for i, podID := range podIDList {
					power := uint64(podDynamicPower[i])
					if err := podEnergy[podID].DynEnergy.AddNewCurr(power); err != nil {
						klog.V(5).Infoln(err)
					}
				}
				klog.V(3).Infof("Get pod powers: %v \n %v from %v (%d x %d)\n", podMetricValues[0:2], podDynamicPower, metricNames, len(podMetricValues), len(podMetricValues[0]))
			}
			for _, v := range podEnergy {
				klog.V(3).Infoln(v)
			}
			klog.V(3).Infoln(nodeEnergy)
			lock.Unlock()
		}
	}()
}

// getAVGCPUFreqAndTotalCPUTime calculates the weighted cpu frequency average
func getAVGCPUFreqAndTotalCPUTime(cpuFrequency map[int32]uint64, cpuTime *[C.CPU_VECTOR_SIZE]uint16) (avgFreq float64, totalCPUTime uint64, activeCPUs []int32) {
	totalFreq := float64(0)
	totalFreqWithoutWeight := float64(0)
	for cpu, freq := range cpuFrequency {
		if int(cpu) > len((*cpuTime))-1 {
			break
		}
		totalCPUTime += uint64(cpuTime[cpu])
		totalFreqWithoutWeight += float64(freq)
	}
	if totalCPUTime == 0 {
		if len(cpuFrequency) == 0 {
			return
		}
		avgFreq = totalFreqWithoutWeight / float64(len(cpuFrequency))
	} else {
		for cpu, freq := range cpuFrequency {
			if int(cpu) > len((*cpuTime))-1 {
				break
			}
			if cpuTime[cpu] != 0 {
				totalFreq += float64(freq) * (float64(cpuTime[cpu]) / float64(totalCPUTime))
				activeCPUs = append(activeCPUs, cpu)
			}
		}
		avgFreq = totalFreqWithoutWeight / float64(len(cpuFrequency))
	}
	return
}

// getActiveCPUs returns active cpu(vcpu) (in case that frequency is not active)
func getActiveCPUs(cpuTime *[C.CPU_VECTOR_SIZE]uint16) (activeCPUs []int32) {
	for cpu := range cpuTime {
		if cpuTime[cpu] != 0 {
			activeCPUs = append(activeCPUs, int32(cpu))
		}
	}
	return
}

// handleInactivePods
func handleInactivePods(foundPod map[string]bool) {
	numOfInactive := len(podEnergy) - len(foundPod)
	if numOfInactive > maxInactivePods {
		alivePods, err := podlister.GetAlivePods()
		if err != nil {
			klog.V(5).Infoln(err)
			return
		}
		for podID := range podEnergy {
			if _, found := alivePods[podID]; !found {
				delete(podEnergy, podID)
			}
		}
	}
}
