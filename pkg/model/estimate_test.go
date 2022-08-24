package model

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sustainable-computing-io/kepler/pkg/config"
)

var (
	MODEL_NAME       string      = "" // auto-select
	LR_NAME          string      = "Linear Regression_10"
	RATIO_MODEL_NAME string      = "CorrRatio"
	METRICS          []string    = []string{"curr_bytes_read", "curr_bytes_writes", "curr_cache_miss", "curr_cgroupfs_cpu_usage_us", "curr_cgroupfs_memory_usage_bytes", "curr_cgroupfs_system_cpu_usage_us", "curr_cgroupfs_user_cpu_usage_us", "curr_cpu_cycles", "curr_cpu_instr", "curr_cpu_time"}
	VALUES           [][]float32 = [][]float32{[]float32{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, []float32{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}}
	FAIL_VALUES      [][]float32 = [][]float32{[]float32{1, 1, 1, 1, 1, 1}}
	empty            []float64   = []float64{}
)

var _ = Describe("Test Estimator Unit", func() {
	It("Get Dynamic Power", func() {
		// should power of each pods
		config.EstimatorModel = MODEL_NAME
		powers := GetDynamicPower(METRICS, VALUES, empty, empty, empty, empty, empty)
		Expect(len(powers)).To(Equal(len(VALUES)))
		config.EstimatorModel = LR_NAME
		powers = GetDynamicPower(METRICS, VALUES, empty, empty, empty, empty, empty)
		Expect(len(powers)).To(Equal(len(VALUES)))
		config.EstimatorModel = RATIO_MODEL_NAME
		powers = GetDynamicPower(METRICS, VALUES, []float64{10, 10}, []float64{5, 5}, []float64{0, 0}, []float64{0, 0}, []float64{0, 0})
		Expect(len(powers)).To(Equal(len(VALUES)))
		fmt.Println(powers)
		// should safely return empty list if fails
		config.EstimatorModel = MODEL_NAME
		powers = GetDynamicPower(METRICS, FAIL_VALUES, empty, empty, empty, empty, empty)
		Expect(len(powers)).To(Equal(0))
	})
	It("Get Ratio Power", func() {
		corePower := []float64{10, 10}
		dramPower := []float64{2, 2}
		uncorePower := []float64{1, 1}
		pkgPower := []float64{15, 15}
		totalCorePower, totalDRAMPower, totalUncorePower, totalPkgPower, _ := GetSumDelta(corePower, dramPower, uncorePower, pkgPower, empty)
		Expect(totalCorePower).Should(BeEquivalentTo(20))
		Expect(totalDRAMPower).Should(BeEquivalentTo(4))
		Expect(totalUncorePower).Should(BeEquivalentTo(2))
		Expect(totalPkgPower).Should(BeEquivalentTo(30))
		sumUsage := GetSumUsageMap(METRICS, VALUES)
		podCore, podDRAM, podUncore, podPkg := GetPowerFromUsageRatio(VALUES, totalCorePower, totalDRAMPower, totalUncorePower, totalPkgPower, sumUsage)
		Expect(len(podCore)).Should(Equal(len(VALUES)))
		Expect(podCore[0]).Should(Equal(podCore[1]))
		Expect(podCore[0]).Should(BeEquivalentTo(10))
		Expect(podDRAM[0]).Should(BeEquivalentTo(2))
		Expect(podUncore[0]).Should(BeEquivalentTo(1))
		Expect(podPkg[0]).Should(BeEquivalentTo(15))
		totalCorePower, totalDRAMPower, totalUncorePower, totalPkgPower, _ = GetSumDelta([]float64{0, 0}, []float64{0, 0}, []float64{0, 0}, pkgPower, empty)
		_, _, _, podPkg = GetPowerFromUsageRatio(VALUES, totalCorePower, totalDRAMPower, totalUncorePower, totalPkgPower, sumUsage)
		Expect(totalCorePower).Should(BeEquivalentTo(0))
		Expect(totalDRAMPower).Should(BeEquivalentTo(0))
		Expect(totalUncorePower).Should(BeEquivalentTo(0))
		Expect(totalPkgPower).Should(BeEquivalentTo(30))
		Expect(podPkg[0]).Should(Equal(podPkg[1]))
		Expect(podPkg[0]).Should(BeEquivalentTo(15))
	})
})
