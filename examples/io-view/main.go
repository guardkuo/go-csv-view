package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

var selector int = 1
var usingBar int = 0

const (
	ldioFilePath   = "ldio.csv"
	pdskioFilePath = "PhdIO0.csv"
	LBALabel       = "LBA"
	DepthInLDLabel = "Depth(LD)"
	DepthLabel     = "Depth"
	ExecTimeLabel  = "ExecTime"
)

func main() {
	if selector == 0 {
		http.HandleFunc("/", renderChartHandler)
	} else {
		http.HandleFunc("/", renderPdskIoChartHandler)
	}
	fmt.Println("伺服器啟動於 http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func renderChartHandler(w http.ResponseWriter, r *http.Request) {
	records, err := loadCsv(ldioFilePath)
	if err != nil {
		http.Error(w, "無法讀取 CSV: "+err.Error(), http.StatusInternalServerError)
		return
	}

	xData, lbaData, depthData := prepareChartData(records)
	line := plotChart(xData, lbaData, depthData)
	line.Render(w)
}

func renderPdskIoChartHandler(w http.ResponseWriter, r *http.Request) {
	records, err := loadCsv(pdskioFilePath)
	if err != nil {
		http.Error(w, "無法讀取 CSV: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if usingBar == 1 {
		xData, lbaData, depthData := preparePdskIOChartData(records)
		line := plotPhDrvIoChart(xData, lbaData, depthData)
		line.Render(w)
	} else {
		xData, lbaData, depthData := preparePdskIOChartLineData(records)
		line := plotPhDrvIoLineChart(xData, lbaData, depthData)
		line.Render(w)
	}
}

func plotChart(xData []string, lbaData []opts.ScatterData, depthData []opts.LineData) *charts.Line {
	mainChart := charts.NewLine()
	mainChart.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Theme: "shine", Width: "100%", Height: "800px",
		}),
		charts.WithTitleOpts(opts.Title{Title: "LD IO View | Infortrend"}),
		charts.WithDataZoomOpts(
			opts.DataZoom{Type: "slider", Start: 0, End: 2},
			opts.DataZoom{Type: "inside"},
		),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "LBA(GB)", Type: "value", Scale: opts.Bool(true)}),
	)

	mainChart.ExtendYAxis(opts.YAxis{Name: "Depth", Type: "value", Scale: opts.Bool(true)})
	mainChart.SetXAxis(xData)

	// --- 1. LBA 散點圖 (Scatter) ---
	lbaScatter := charts.NewScatter()
	lbaScatter.AddSeries(LBALabel, lbaData).
		SetSeriesOptions(
			// 關鍵修正：透過匿名函式直接操作 SingleSeries 內部，但避開 Symbol 欄位
			// 我們改用 Type 強制指定為散點，ECharts 預設就會用 circle 且不畫線
			func(s *charts.SingleSeries) {
				s.Type = "scatter"
				s.SymbolSize = 4 // 將 size 從 2 縮小為 1
			},
			charts.WithItemStyleOpts(opts.ItemStyle{Opacity: opts.Float(0.7)}),
		)

	// --- 2. Depth 折線圖 (Line) ---
	depthLine := charts.NewLine()
	depthLine.AddSeries(DepthInLDLabel, depthData).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{
				YAxisIndex: 1,
				Smooth:     opts.Bool(false),
				// 在 LineChart 內部結構使用 Symbol 不會產生歧義
				Symbol: "none",
			}),
			func(s *charts.SingleSeries) {
				// 使用匿名函數強制開啟 Large 模式，這是解決殘影的核心
				s.Large = opts.Bool(true)
				s.LargeThreshold = 2000 // 超過 2000 點就啟動優化
			},
		)

	// --- 3. 疊加 ---
	mainChart.Overlap(lbaScatter)
	mainChart.Overlap(depthLine)

	return mainChart
}

func plotPhDrvIoChart(xData []string, ExecTimeData []opts.ScatterData, depthData []opts.BarData) *charts.Line {
	mainChart := charts.NewLine()
	mainChart.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Theme: "shine", Width: "100%", Height: "800px",
		}),
		charts.WithTitleOpts(opts.Title{Title: "Pdsk IO View | Infortrend"}),
		charts.WithDataZoomOpts(
			opts.DataZoom{Type: "slider", Start: 0, End: 2},
			opts.DataZoom{Type: "inside"},
		),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Time(us)", Type: "value", Scale: opts.Bool(true)}),
	)

	mainChart.ExtendYAxis(opts.YAxis{Name: "Depth", Type: "value", Scale: opts.Bool(true)})
	mainChart.SetXAxis(xData)

	// --- 1. LBA 散點圖 (Scatter) ---
	ExecTimeScatter := charts.NewScatter()
	ExecTimeScatter.AddSeries(ExecTimeLabel, ExecTimeData).
		SetSeriesOptions(
			// 關鍵修正：透過匿名函式直接操作 SingleSeries 內部，但避開 Symbol 欄位
			// 我們改用 Type 強制指定為散點，ECharts 預設就會用 circle 且不畫線
			func(s *charts.SingleSeries) {
				s.Type = "scatter"
				s.SymbolSize = 4 // 將 size 從 2 縮小為 1
			},
			charts.WithItemStyleOpts(opts.ItemStyle{Opacity: opts.Float(0.7)}),
		)

	bar := charts.NewBar()
	bar.AddSeries(DepthLabel, depthData).
		SetSeriesOptions(
			func(s *charts.SingleSeries) {
				s.YAxisIndex = 1
				s.ShowBackground = opts.Bool(false)
				// 使用匿名函數強制開啟 Large 模式，這是解決殘影的核心
				s.Large = opts.Bool(true)
				s.LargeThreshold = 2000 // 超過 2000 點就啟動優化
			},
		)

	// --- 3. 疊加 ---
	mainChart.Overlap(ExecTimeScatter)
	mainChart.Overlap(bar)

	return mainChart
}

func plotPhDrvIoLineChart(xData []string, ExecTimeData []opts.ScatterData, depthData []opts.LineData) *charts.Line {
	mainChart := charts.NewLine()
	mainChart.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Theme: "shine", Width: "100%", Height: "800px",
		}),
		charts.WithTitleOpts(opts.Title{Title: "Pdsk IO View | Infortrend"}),
		charts.WithDataZoomOpts(
			opts.DataZoom{Type: "slider", Start: 0, End: 2},
			opts.DataZoom{Type: "inside"},
		),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Time(us)", Type: "value", Scale: opts.Bool(true)}),
	)

	mainChart.ExtendYAxis(opts.YAxis{Name: "Depth", Type: "value", Scale: opts.Bool(true)})
	mainChart.SetXAxis(xData)

	// --- 1. LBA 散點圖 (Scatter) ---
	ExecTimeScatter := charts.NewScatter()
	ExecTimeScatter.AddSeries(ExecTimeLabel, ExecTimeData).
		SetSeriesOptions(
			// 關鍵修正：透過匿名函式直接操作 SingleSeries 內部，但避開 Symbol 欄位
			// 我們改用 Type 強制指定為散點，ECharts 預設就會用 circle 且不畫線
			func(s *charts.SingleSeries) {
				s.Type = "scatter"
				s.SymbolSize = 4 // 將 size 從 2 縮小為 1
			},
			charts.WithItemStyleOpts(opts.ItemStyle{Opacity: opts.Float(0.7)}),
		)

	// --- 2. Depth 折線圖 (Line) ---
	depthLine := charts.NewLine()
	depthLine.AddSeries(DepthLabel, depthData).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{
				YAxisIndex: 1,
				Smooth:     opts.Bool(false),
				// 在 LineChart 內部結構使用 Symbol 不會產生歧義
				Symbol: "none",
				Step:   "start",
			}),
			func(s *charts.SingleSeries) {
				// 使用匿名函數強制開啟 Large 模式，這是解決殘影的核心
				s.Large = opts.Bool(true)
				s.LargeThreshold = 2000 // 超過 2000 點就啟動優化
			},
		)

	// --- 3. 疊加 ---
	mainChart.Overlap(ExecTimeScatter)
	mainChart.Overlap(depthLine)

	return mainChart
}

func prepareChartData(records [][]string) ([]string, []opts.ScatterData, []opts.LineData) {
	startRow := 0
	if len(records) > 0 && strings.Contains(records[0][0], "LD") {
		startRow = 1
	}

	x := make([]string, 0)
	lba := make([]opts.ScatterData, 0)
	depth := make([]opts.LineData, 0)
	var startTestTime uint64

	for _, record := range records[startRow:] {
		if len(record) < 7 {
			continue
		}

		startTVal, _ := strconv.ParseUint(record[1], 10, 64)
		if startTestTime == 0 {
			startTestTime = startTVal
			startTVal = 0
		} else {
			startTVal = startTVal - startTestTime
		}

		lbaVal, _ := strconv.ParseUint(record[4], 10, 64)
		depthVal, _ := strconv.ParseUint(record[6], 10, 64)

		x = append(x, strconv.FormatUint(startTVal, 10))
		lba = append(lba, opts.ScatterData{Value: lbaVal >> 21})
		depth = append(depth, opts.LineData{Value: depthVal})
	}
	return x, lba, depth
}

func preparePdskIOChartData(records [][]string) ([]string, []opts.ScatterData, []opts.BarData) {
	startRow := 0
	if len(records) > 0 && strings.Contains(records[0][0], "Start Time") {
		startRow = 1
	}

	x := make([]string, 0)
	execT := make([]opts.ScatterData, 0)
	depth := make([]opts.BarData, 0)
	var startTestTime uint64

	for _, record := range records[startRow:] {
		if len(record) < 6 {
			continue
		}
		endTVal, _ := strconv.ParseUint(record[1], 10, 64)
		if endTVal == 0 {
			continue
		}
		startTVal, _ := strconv.ParseUint(record[0], 10, 64)
		if startTestTime == 0 {
			startTestTime = startTVal
			startTVal = 0
		} else {
			startTVal = (startTVal - startTestTime)
		}

		ExecTimeVal, _ := strconv.ParseUint(record[5], 10, 64)
		if ExecTimeVal == startTVal {
			continue
		}

		depthVal, _ := strconv.ParseUint(record[4], 10, 64)

		x = append(x, strconv.FormatUint(startTVal>>2, 10))
		execT = append(execT, opts.ScatterData{Value: ExecTimeVal >> 2})
		depth = append(depth, opts.BarData{Value: depthVal})
	}
	return x, execT, depth
}

func preparePdskIOChartLineData(records [][]string) ([]string, []opts.ScatterData, []opts.LineData) {
	startRow := 0
	if len(records) > 0 && strings.Contains(records[0][0], "Start Time") {
		startRow = 1
	}

	x := make([]string, 0)
	execT := make([]opts.ScatterData, 0)
	depth := make([]opts.LineData, 0)
	var startTestTime uint64

	for _, record := range records[startRow:] {
		if len(record) < 6 {
			continue
		}
		endTVal, _ := strconv.ParseUint(record[1], 10, 64)
		if endTVal == 0 {
			continue
		}
		startTVal, _ := strconv.ParseUint(record[0], 10, 64)
		if startTestTime == 0 {
			startTestTime = startTVal
			startTVal = 0
		} else {
			startTVal = (startTVal - startTestTime)
		}

		ExecTimeVal, _ := strconv.ParseUint(record[5], 10, 64)
		if ExecTimeVal == startTVal {
			continue
		}

		depthVal, _ := strconv.ParseUint(record[4], 10, 64)

		x = append(x, strconv.FormatUint(startTVal>>2, 10))
		execT = append(execT, opts.ScatterData{Value: ExecTimeVal >> 2})
		depth = append(depth, opts.LineData{Value: depthVal})
	}
	return x, execT, depth
}

func loadCsv(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	return reader.ReadAll()
}
